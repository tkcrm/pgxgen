package schema

import (
	"fmt"
	"path/filepath"

	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/sqlparser"
	"github.com/tkcrm/pgxgen/internal/sqlparser/catalog"
)

// CatalogResult holds the parsed catalog with associated config metadata.
type CatalogResult struct {
	OutputDir     string
	SchemaDir     []string
	QueriesDir    []string
	GoPackageName string
	Catalog       *catalog.Catalog
}

// ISchema provides access to parsed database schema catalogs.
type ISchema interface {
	GetSchema(sqlcConfig config.Sqlc, sqlcConfigDir, schemaDir string) (CatalogResult, error)
}

type schema struct {
	catalogs map[string]CatalogResult
}

func New() ISchema {
	return &schema{
		catalogs: make(map[string]CatalogResult),
	}
}

func (s *schema) GetSchema(sqlcConfig config.Sqlc, sqlcConfigDir, schemaDir string) (CatalogResult, error) {
	resolvedSchemaDir := resolveRelativePath(sqlcConfigDir, schemaDir)

	absSchemaDir, err := filepath.Abs(resolvedSchemaDir)
	if err != nil {
		return CatalogResult{}, fmt.Errorf("abs path for schema dir: %w", err)
	}

	if item, ok := s.catalogs[absSchemaDir]; ok {
		return item, nil
	}

	paths := sqlcConfig.GetPaths()

	// Find the matching entry in sqlc config
	for i, sp := range paths.SchemaPaths {
		resolvedSP := resolveRelativePath(sqlcConfigDir, sp)
		absSP, err := filepath.Abs(resolvedSP)
		if err != nil {
			continue
		}
		if absSP != absSchemaDir {
			continue
		}

		engine := "postgresql"
		if i < len(paths.Engines) && paths.Engines[i] != "" {
			engine = paths.Engines[i]
		}

		parser, err := sqlparser.NewParser(engine)
		if err != nil {
			return CatalogResult{}, fmt.Errorf("create parser for engine %s: %w", engine, err)
		}

		files, err := sqlparser.ResolveSchemaFiles(resolvedSchemaDir)
		if err != nil {
			return CatalogResult{}, fmt.Errorf("resolve schema files: %w", err)
		}

		cat, err := parser.ParseSchema(files)
		if err != nil {
			return CatalogResult{}, fmt.Errorf("parse schema: %w", err)
		}

		result := CatalogResult{
			OutputDir:  resolveRelativePath(sqlcConfigDir, paths.OutPaths[i]),
			SchemaDir:  []string{resolvedSP},
			QueriesDir: []string{resolveRelativePath(sqlcConfigDir, paths.QueriesPaths[i])},
			Catalog:    cat,
		}

		s.catalogs[absSchemaDir] = result
		return result, nil
	}

	return CatalogResult{}, fmt.Errorf("cannot find catalog for schema dir: %s", schemaDir)
}

// GetCatalogs parses all schemas from the sqlc config and returns a list of catalog results.
// sqlcConfigDir is the directory containing sqlc.yaml — all relative paths in the config
// are resolved relative to it.
func GetCatalogs(sqlcConfig config.Sqlc, sqlcConfigDir string) ([]CatalogResult, error) {
	paths := sqlcConfig.GetPaths()
	var results []CatalogResult

	for i, sp := range paths.SchemaPaths {
		engine := "postgresql"
		if i < len(paths.Engines) && paths.Engines[i] != "" {
			engine = paths.Engines[i]
		}

		parser, err := sqlparser.NewParser(engine)
		if err != nil {
			return nil, fmt.Errorf("create parser for engine %s: %w", engine, err)
		}

		resolvedSP := resolveRelativePath(sqlcConfigDir, sp)
		files, err := sqlparser.ResolveSchemaFiles(resolvedSP)
		if err != nil {
			return nil, fmt.Errorf("resolve schema files for %s: %w", sp, err)
		}

		cat, err := parser.ParseSchema(files)
		if err != nil {
			return nil, fmt.Errorf("parse schema %s: %w", sp, err)
		}

		results = append(results, CatalogResult{
			OutputDir:  resolveRelativePath(sqlcConfigDir, paths.OutPaths[i]),
			SchemaDir:  []string{resolvedSP},
			QueriesDir: []string{resolveRelativePath(sqlcConfigDir, paths.QueriesPaths[i])},
			Catalog:    cat,
		})
	}

	return results, nil
}

// resolveRelativePath joins baseDir and p if p is not absolute.
func resolveRelativePath(baseDir, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(baseDir, p)
}

// GetCatalogByOutputDir finds a catalog result by its output directory path.
func GetCatalogByOutputDir(catalogs []CatalogResult, outputDir string) (CatalogResult, error) {
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return CatalogResult{}, fmt.Errorf("abs path: %w", err)
	}

	for _, c := range catalogs {
		absOut, err := filepath.Abs(c.OutputDir)
		if err != nil {
			continue
		}
		if absOut == absOutputDir {
			return c, nil
		}
	}

	return CatalogResult{}, fmt.Errorf("cannot find catalog for output dir: %s", outputDir)
}
