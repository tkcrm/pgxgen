package goconstatnts

import (
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/tkcrm/modules/pkg/templates"
	"github.com/tkcrm/pgxgen/internal/assets"
	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/schema"
	"github.com/tkcrm/pgxgen/utils"
)

type IGoConstants interface {
	GenerateConstants() error
}

type goConstants struct {
	config config.Config
	schema schema.ISchema
}

func New(config config.Config) IGoConstants {
	return &goConstants{
		config: config,
		schema: schema.New(),
	}
}

const defaultConstatsFileName = "constants_gen.go"

type GenTableNamesParamsItem struct {
	TableNamePreffix string
	TableName        string
}
type ConstantParamsItem struct {
	Package string
	Version string
	Tables  []GenTableNamesParamsItem
}

func (s *goConstants) GenerateConstants() error {
	if len(s.config.Pgxgen.GoConstants.Tables) == 0 {
		return nil
	}

	var params generateConstantsParams

	for tableName, table := range s.config.Pgxgen.GoConstants.Tables {
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
				schemaDir = s.config.Sqlc.GetPaths().MigrationsPaths[index]
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
				if err := params.addConstantItem(s.config.Pgxgen.Version, table.OutputDir, tableName); err != nil {
					return err
				}
			}
		}
	}

	for outputDir, item := range params.ConstantsParams {
		tpl := templates.New()
		tpl.AddFunc("isNotEmptyArray", func(arr []GenTableNamesParamsItem) bool {
			return len(arr) > 0
		})

		compiledRes, err := tpl.Compile(templates.CompileParams{
			TemplateName: "crudConstants",
			TemplateType: templates.TextTemplateType,
			FS:           assets.TemplatesFS,
			FSPaths: []string{
				"templates/crud-constants.go.tmpl",
			},
			Data: item,
		})
		if err != nil {
			return errors.Wrap(err, "tpl.Compile error")
		}

		compiledRes, err = utils.UpdateGoImports(compiledRes)
		if err != nil {
			return errors.Wrap(err, "UpdateGoImports error")
		}

		if err := utils.SaveFile(outputDir, defaultConstatsFileName, compiledRes); err != nil {
			return errors.Wrap(err, "SaveFile error")
		}
	}

	return nil
}
