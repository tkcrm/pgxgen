package codegen

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tkcrm/pgxgen/internal/codegen/constants"
	"github.com/tkcrm/pgxgen/internal/codegen/crud"
	"github.com/tkcrm/pgxgen/internal/codegen/models"
	sqlcgen "github.com/tkcrm/pgxgen/internal/codegen/sqlc"
	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/engine"
	"github.com/tkcrm/pgxgen/internal/sqlparser"
	"github.com/tkcrm/pgxgen/internal/sqlparser/catalog"
	"github.com/tkcrm/pgxgen/pkg/logger"
)

// GenerateOpts controls what to generate.
type GenerateOpts struct {
	DryRun  bool
	Debug   bool
	Targets []string // crud, models, constants, sqlc, all (empty = all)
}

// Orchestrator runs all generators in correct order.
type Orchestrator struct {
	logger    logger.Logger
	cfg       *config.V2Config
	configDir string
}

// NewOrchestrator creates a new orchestrator.
func NewOrchestrator(l logger.Logger, cfg *config.V2Config, configPath string) *Orchestrator {
	return &Orchestrator{
		logger:    l,
		cfg:       cfg,
		configDir: filepath.Dir(configPath),
	}
}

func (o *Orchestrator) shouldRun(target string, opts GenerateOpts) bool {
	if len(opts.Targets) == 0 {
		return true
	}
	for _, t := range opts.Targets {
		if t == "all" || t == target {
			return true
		}
	}
	return false
}

// Generate runs all configured generators.
func (o *Orchestrator) Generate(ctx context.Context, opts GenerateOpts) ([]GenerationResult, error) {
	var allResults []GenerationResult

	for _, schema := range o.cfg.Schemas {
		start := time.Now()
		o.logger.Infof("generating for schema %q (engine: %s)", schema.Name, schema.Engine)

		if o.cfg.Templates != nil && o.cfg.Templates.CrudDir != "" {
			crud.SetCustomTemplateDir(filepath.Join(o.configDir, o.cfg.Templates.CrudDir))
		}

		debugf := func(format string, args ...interface{}) {
			if opts.Debug {
				o.logger.Infof("[debug] "+format, args...)
			}
		}

		// Parse schema ONCE — reuse for all generators
		var cat *catalog.Catalog
		needsParsing := o.shouldRun("crud", opts) || o.shouldRun("constants", opts) || o.shouldRun("models", opts)
		if needsParsing {
			stepStart := time.Now()
			var err error
			cat, err = o.parseSchema(&schema)
			if err != nil {
				return nil, fmt.Errorf("parse schema %s: %w", schema.Name, err)
			}
			debugf("parse schema: %s", time.Since(stepStart))
		}

		// 1. Generate CRUD SQL
		if o.shouldRun("crud", opts) && cat != nil {
			stepStart := time.Now()
			results, err := o.generateCrud(&schema, cat)
			if err != nil {
				return nil, fmt.Errorf("generate crud for schema %s: %w", schema.Name, err)
			}
			allResults = append(allResults, results...)
			debugf("generate crud: %s (%d files)", time.Since(stepStart), len(results))
		}

		// 1b. Generate custom queries
		if o.shouldRun("crud", opts) && len(schema.CustomQueries) > 0 {
			stepStart := time.Now()
			results, err := o.generateCustomQueries(&schema)
			if err != nil {
				return nil, fmt.Errorf("generate custom queries for schema %s: %w", schema.Name, err)
			}
			allResults = append(allResults, results...)
			debugf("generate custom queries: %s (%d files)", time.Since(stepStart), len(results))
		}

		// Flush CRUD + custom-query results to disk now so sqlc can read them.
		// sqlc parses query files from disk, so they must exist before sqlc runs.
		if err := WriteResults(allResults, opts.DryRun); err != nil {
			return nil, fmt.Errorf("write crud results for schema %s: %w", schema.Name, err)
		}
		for i := range allResults {
			allResults[i].Action = ActionUnchanged
		}

		// 2. Generate models (reuses parsed catalog + sqlc config for overrides/defaults)
		if o.shouldRun("models", opts) && schema.Models != nil {
			stepStart := time.Now()
			if err := models.GenerateFromV2Config(
				o.logger, o.configDir, schema.Models, schema.Engine, schema.SchemaDir, cat, schema.Sqlc,
			); err != nil {
				return nil, fmt.Errorf("generate models for schema %s: %w", schema.Name, err)
			}
			debugf("generate models: %s", time.Since(stepStart))
		}

		// 3. Run sqlc — skipped in dry-run because sqlc has no preview mode and
		// its inputs (CRUD SQL) were not written to disk.
		if !opts.DryRun && o.shouldRun("sqlc", opts) && schema.Sqlc != nil {
			stepStart := time.Now()
			gen := sqlcgen.NewGenerator(o.logger, o.configDir)
			if err := gen.Run(&schema); err != nil {
				return nil, fmt.Errorf("run sqlc for schema %s: %w", schema.Name, err)
			}
			debugf("sqlc generate + post-process: %s", time.Since(stepStart))
		}

		// 4. Generate constants (reuse parsed catalog)
		if o.shouldRun("constants", opts) && cat != nil {
			stepStart := time.Now()
			if err := constants.GenerateFromV2ConfigWithCatalog(
				o.logger, o.configDir, &schema, cat,
			); err != nil {
				return nil, fmt.Errorf("generate constants for schema %s: %w", schema.Name, err)
			}
			debugf("generate constants: %s", time.Since(stepStart))
		}

		o.logger.Infof("schema %q generated in %s", schema.Name, time.Since(start))
	}

	return allResults, nil
}

// parseSchema parses SQL files once and returns the catalog.
func (o *Orchestrator) parseSchema(schema *config.SchemaConfig) (*catalog.Catalog, error) {
	schemaDir := o.resolvePath(schema.SchemaDir)

	parser, err := sqlparser.NewParser(schema.Engine)
	if err != nil {
		return nil, fmt.Errorf("create parser: %w", err)
	}

	files, err := sqlparser.ResolveSchemaFiles(schemaDir)
	if err != nil {
		return nil, fmt.Errorf("resolve schema files from %s: %w", schemaDir, err)
	}

	cat, err := parser.ParseSchema(files)
	if err != nil {
		return nil, fmt.Errorf("parse schema: %w", err)
	}

	return cat, nil
}

func (o *Orchestrator) generateCrud(schema *config.SchemaConfig, cat *catalog.Catalog) ([]GenerationResult, error) {
	if len(schema.Tables) == 0 {
		return nil, nil
	}

	eng, err := engine.New(schema.Engine)
	if err != nil {
		return nil, err
	}

	gen := crud.New(eng)

	tableNames := make([]string, 0, len(schema.Tables))
	for name := range schema.Tables {
		tableNames = append(tableNames, name)
	}
	sort.Strings(tableNames)

	var results []GenerationResult

	for _, tableName := range tableNames {
		tableConfig := schema.Tables[tableName]

		columns := crud.GetTableColumns(cat, tableName)
		if columns == nil {
			return nil, fmt.Errorf("table %q not found in schema", tableName)
		}

		var defaultCrud *config.CrudDefaultsConfig
		if schema.Defaults != nil {
			defaultCrud = schema.Defaults.Crud
		}

		data, err := gen.GenerateTable(tableName, tableConfig, defaultCrud, columns)
		if err != nil {
			return nil, fmt.Errorf("generate table %s: %w", tableName, err)
		}

		if len(data) == 0 {
			continue
		}

		queriesDir := schema.ResolveQueriesDir(tableName)
		if tableConfig.QueriesDir != "" {
			queriesDir = tableConfig.QueriesDir
		}
		if queriesDir == "" {
			return nil, fmt.Errorf("no queries_dir resolved for table %s", tableName)
		}

		outputPath := filepath.Join(o.resolvePath(queriesDir), tableName+"_gen.sql")
		results = append(results, PrepareResult(outputPath, data))
	}

	return results, nil
}

func (o *Orchestrator) generateCustomQueries(schema *config.SchemaConfig) ([]GenerationResult, error) {
	var results []GenerationResult

	for _, cq := range schema.CustomQueries {
		content := fmt.Sprintf("-- name: %s :%s\n%s\n", cq.Name, cq.Type, cq.SQL)

		var queriesDir string
		if cq.OutputDir != "" {
			queriesDir = cq.OutputDir
		} else if cq.Table != "" {
			queriesDir = schema.ResolveQueriesDir(cq.Table)
		}
		if queriesDir == "" {
			return nil, fmt.Errorf("custom query %q: no output_dir or table specified", cq.Name)
		}

		fileName := fmt.Sprintf("custom_%s_gen.sql", strings.ToLower(cq.Name))
		outputPath := filepath.Join(o.resolvePath(queriesDir), fileName)
		results = append(results, PrepareResult(outputPath, []byte(content)))
	}

	return results, nil
}

func (o *Orchestrator) resolvePath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(o.configDir, p)
}
