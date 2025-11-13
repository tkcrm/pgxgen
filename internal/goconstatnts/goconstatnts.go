package goconstatnts

import (
	"bytes"
	"context"
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
	sqlcAbsFilePath, err := filepath.Abs(s.config.ConfigPaths.SqlcConfigFilePath)
	if err != nil {
		return fmt.Errorf("failed to get sqlc config abs file path: %w", err)
	}

	sqlcDir := filepath.Dir(sqlcAbsFilePath)

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
			var schemaDir string
			for index, path := range s.config.Sqlc.GetPaths().OutPaths {
				absPath1, err := filepath.Abs(table.OutputDir)
				if err != nil {
					return fmt.Errorf("failed to get absolute path: %w", err)
				}

				absPath2, err := filepath.Abs(path)
				if err != nil {
					return fmt.Errorf("failed to get absolute path: %w", err)
				}
				if absPath1 == absPath2 {
					schemaDir = s.config.Sqlc.GetPaths().SchemaPaths[index]
					break
				}
			}

			schemaDir = filepath.Join(sqlcDir, schemaDir)

			if schemaDir == "" {
				return fmt.Errorf("can not find schema dir for output dir: %s", table.OutputDir)
			}

			catalog, err := s.schema.GetSchema(s.config.ConfigPaths.SqlcConfigFilePath, schemaDir)
			if err != nil {
				return fmt.Errorf("failed to get schema: %w", err)
			}

			for _, schema := range catalog.Catalog.Schemas {
				for _, t := range schema.Tables {
					if t.Rel.Name != tableName {
						continue
					}

					columnNames := make([]string, 0, len(t.Columns))
					if table.IncludeColumnNames {
						for _, column := range t.Columns {
							columnNames = append(columnNames, column.Name)
						}
					}

					outputDir := filepath.Join(sqlcDir, table.OutputDir)

					if err := params.addConstantItem(s.config.Pgxgen.Version, outputDir, tableName, columnNames); err != nil {
						return fmt.Errorf("failed to add constant item: %w", err)
					}
				}
			}
		}

		for outputDir, item := range params.ConstantsParams {
			buf := new(bytes.Buffer)
			if err := templates.Constatnts(item).Render(context.Background(), buf); err != nil {
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
