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
	for _, cfg := range s.config.Pgxgen.Sqlc {
		if len(cfg.GoConstants.Tables) == 0 {
			return nil
		}

		if err := cfg.Validate(); err != nil {
			return err
		}

		s.logger.Infof("generate constants for schema: %s", cfg.SchemaDir)
		timeStart := time.Now()

		var params generateConstantsParams

		for tableName, table := range cfg.GoConstants.Tables {
			var schemaDir string
			for index, path := range s.config.Sqlc.GetPaths().OutPaths {
				absPath1, err := filepath.Abs(table.OutputDir)
				if err != nil {
					return err
				}

				absPath2, err := filepath.Abs(path)
				if err != nil {
					return err
				}
				if absPath1 == absPath2 {
					schemaDir = s.config.Sqlc.GetPaths().SchemaPaths[index]
					break
				}
			}
			if schemaDir == "" {
				return fmt.Errorf("can not find output dir for path: %s", table.OutputDir)
			}

			catalog, err := s.schema.GetSchema(schemaDir)
			if err != nil {
				return err
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

					if err := params.addConstantItem(s.config.Pgxgen.Version, table.OutputDir, tableName, columnNames); err != nil {
						return err
					}
				}
			}
		}

		for outputDir, item := range params.ConstantsParams {
			buf := new(bytes.Buffer)
			if err := templates.Constatnts(item).Render(context.Background(), buf); err != nil {
				return err
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
