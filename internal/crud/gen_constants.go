package crud

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/gobeam/stringy"
	"github.com/pkg/errors"
	"github.com/tkcrm/modules/pkg/templates"
	cmnutils "github.com/tkcrm/modules/pkg/utils"
	"github.com/tkcrm/pgxgen/internal/assets"
	"github.com/tkcrm/pgxgen/utils"
)

const defaultConstatsFileName = "constants_gen.go"

type generateConstantsParams struct {
	ConstantsParams map[string]ConstantParamsItem
}

func (s *generateConstantsParams) addConstantItem(version, outputDir, tableName string) error {
	if s.ConstantsParams == nil {
		s.ConstantsParams = make(map[string]ConstantParamsItem)
	}

	packageName, err := utils.GetGoPackageNameForDir(outputDir)
	if err != nil {
		return errors.Wrap(err, "GetGoPackageNameForDir error")
	}

	params, ok := s.ConstantsParams[outputDir]
	if !ok {
		params = ConstantParamsItem{
			Package: packageName,
			Version: version,
		}
	}

	if _, ok := cmnutils.FindInArray(params.Tables, func(v GenTableNamesParamsItem) bool {
		return v.TableName == tableName
	}); !ok {
		re := regexp.MustCompile(`[\_\-0-9]`)
		tableNamePrffix := re.ReplaceAllString(tableName, " ")
		tableNamePrffix = stringy.New(tableNamePrffix).CamelCase()
		tableNamePrffix = stringy.New(tableNamePrffix).UcFirst()

		params.Tables = append(params.Tables, GenTableNamesParamsItem{
			TableNamePreffix: tableNamePrffix,
			TableName:        tableName,
		})
	}

	sort.Slice(params.Tables, func(i, j int) bool {
		return params.Tables[i].TableNamePreffix < params.Tables[j].TableNamePreffix
	})

	s.ConstantsParams[outputDir] = params

	return nil
}

type GenTableNamesParamsItem struct {
	TableNamePreffix string
	TableName        string
}
type ConstantParamsItem struct {
	Package string
	Version string
	Tables  []GenTableNamesParamsItem
}

func (s *crud) GenerateConstants() error {
	if !s.config.Pgxgen.CrudParams.GenerateTableNames {
		return nil
	}

	res, err := s.process()
	if err != nil {
		return errors.Wrap(err, "process error")
	}

	var params generateConstantsParams

	tableNamePaths := make(map[string][]string, len(res))
	for tableName := range res {
		tableParams, ok := s.config.Pgxgen.CrudParams.Tables[tableName]
		if !ok {
			return fmt.Errorf("can not find table params for table: %s", tableName)
		}

		if tableParams.OutputDir != "" {
			tableNamePaths[tableParams.OutputDir] = append(tableNamePaths[tableParams.OutputDir], tableName)
			continue
		}

		for _, p := range s.config.Sqlc.GetPaths().QueriesPaths {
			tableNamePaths[p] = append(tableNamePaths[p], tableName)
		}
	}

	for index, path := range s.config.Sqlc.GetPaths().QueriesPaths {
		outPath := s.config.Sqlc.GetPaths().OutPaths[index]
		tableNames, ok := tableNamePaths[path]
		if !ok {
			return fmt.Errorf("can not find table name in paths")
		}

		for _, tableName := range tableNames {
			if err := params.addConstantItem(s.config.Pgxgen.Version, outPath, tableName); err != nil {
				return err
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
