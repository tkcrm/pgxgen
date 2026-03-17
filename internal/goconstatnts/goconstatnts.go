package goconstatnts

import (
	"bytes"
	"fmt"
	"path/filepath"
	"time"

	"github.com/tkcrm/pgxgen/internal/assets/templates"
	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/schema"
	"github.com/tkcrm/pgxgen/pkg/logger"
	"github.com/tkcrm/pgxgen/utils"
)

type IGoConstants interface {
	GenerateConstants() error
}

type goConstants struct {
	logger logger.Logger
	config config.Config
	schema schema.ISchema
}

func New(logger logger.Logger, config config.Config) IGoConstants {
	return &goConstants{
		logger: logger,
		config: config,
		schema: schema.New(),
	}
}

const defaultConstatsFileName = "constants_gen.go"

func (s *goConstants) GenerateConstants() error {
	sqlcConfigDir := filepath.Dir(s.config.ConfigPaths.SqlcConfigFilePath)
	pgxgenConfigDir := filepath.Dir(s.config.ConfigPaths.PgxgenConfigFilePath)

	for _, cfg := range s.config.Pgxgen.Sqlc {
		if len(cfg.GoConstants.Tables) == 0 {
			return nil
		}

		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("validation error: %w", err)
		}

		s.logger.Infof("generate constants for schema: %s", cfg.SchemaDir)
		timeStart := time.Now()

		var params generateConstantsParams

		for tableName, table := range cfg.GoConstants.Tables {
			// Resolve table.OutputDir relative to pgxgen config dir
			resolvedOutputDir := table.OutputDir
			if !filepath.IsAbs(resolvedOutputDir) {
				resolvedOutputDir = filepath.Join(pgxgenConfigDir, resolvedOutputDir)
			}

			var schemaDir string
			for index, path := range s.config.Sqlc.GetPaths().OutPaths {
				// Resolve sqlc out path relative to sqlc config dir
				resolvedPath := path
				if !filepath.IsAbs(resolvedPath) {
					resolvedPath = filepath.Join(sqlcConfigDir, resolvedPath)
				}

				absPath1, err := filepath.Abs(resolvedOutputDir)
				if err != nil {
					return fmt.Errorf("failed to get absolute path: %w", err)
				}

				absPath2, err := filepath.Abs(resolvedPath)
				if err != nil {
					return fmt.Errorf("failed to get absolute path: %w", err)
				}
				if absPath1 == absPath2 {
					schemaDir = s.config.Sqlc.GetPaths().SchemaPaths[index]
					break
				}
			}

			if schemaDir == "" {
				return fmt.Errorf("can not find schema dir for output dir: %s", table.OutputDir)
			}

			catalog, err := s.schema.GetSchema(s.config.Sqlc, sqlcConfigDir, schemaDir)
			if err != nil {
				return fmt.Errorf("failed to get schema: %w", err)
			}

			for _, schema := range catalog.Catalog.Schemas {
				for _, t := range schema.Tables {
					if t.Name != tableName {
						continue
					}

					columnNames := make([]string, 0, len(t.Columns))
					if table.IncludeColumnNames {
						for _, column := range t.Columns {
							columnNames = append(columnNames, column.Name)
						}
					}

					absOutputDir, err := filepath.Abs(resolvedOutputDir)
					if err != nil {
						return fmt.Errorf("failed to get absolute path: %w", err)
					}

					if err := params.addConstantItem(s.config.Pgxgen.Version, absOutputDir, tableName, columnNames); err != nil {
						return fmt.Errorf("failed to add constant item: %w", err)
					}
				}
			}
		}

		for outputDir, item := range params.ConstantsParams {
			buf := new(bytes.Buffer)
			if err := templates.RenderConstants(item, buf); err != nil {
				return fmt.Errorf("failed to render constants template: %w", err)
			}

			compiledRes, err := utils.UpdateGoImports(buf.Bytes())
			if err != nil {
				return fmt.Errorf("UpdateGoImports error: %w", err)
			}

			if err := utils.SaveFile(outputDir, defaultConstatsFileName, compiledRes); err != nil {
				return fmt.Errorf("SaveFile error: %w", err)
			}
		}

		s.logger.Infof("constants successfully generated in: %s", time.Since(timeStart))
	}

	return nil
}
