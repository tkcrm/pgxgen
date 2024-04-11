package goconstatnts

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/gobeam/stringy"
	cmnutils "github.com/tkcrm/modules/pkg/utils"
	"github.com/tkcrm/pgxgen/internal/assets/templates"
	"github.com/tkcrm/pgxgen/utils"
)

type generateConstantsParams struct {
	ConstantsParams map[string]templates.ConstantsParams
}

func (s *generateConstantsParams) addConstantItem(version, outputDir, tableName string, columnNames []string) error {
	if s.ConstantsParams == nil {
		s.ConstantsParams = make(map[string]templates.ConstantsParams)
	}

	packageName, err := utils.GetGoPackageNameForDir(outputDir)
	if err != nil {
		return fmt.Errorf("GetGoPackageNameForDir error: %w", err)
	}

	params, ok := s.ConstantsParams[outputDir]
	if !ok {
		params = templates.ConstantsParams{
			Package: packageName,
			Version: version,
		}
	}

	if _, ok := cmnutils.FindInArray(params.Tables, func(v templates.ConstantsTableNamesParamsItem) bool {
		return v.Name == tableName
	}); !ok {
		re := regexp.MustCompile(`[\_\-0-9]`)
		tableNamePrffix := re.ReplaceAllString(tableName, " ")
		tableNamePrffix = stringy.New(tableNamePrffix).CamelCase()
		tableNamePrffix = stringy.New(tableNamePrffix).UcFirst()

		params.Tables = append(params.Tables, templates.ConstantsTableNamesParamsItem{
			NamePreffix: tableNamePrffix,
			Name:        tableName,
		})

		if len(columnNames) > 0 {
			for _, columnName := range columnNames {
				columnNamePrffix := re.ReplaceAllString(tableName+"_"+columnName, " ")
				columnNamePrffix = stringy.New(columnNamePrffix).CamelCase()
				columnNamePrffix = stringy.New(columnNamePrffix).UcFirst()

				params.ColumnNames = append(params.ColumnNames, templates.ConstantsColumnNamesParamsItem{
					TableName:   tableName,
					NamePreffix: columnNamePrffix,
					Name:        columnName,
				})
			}
		}
	}

	sort.Slice(params.Tables, func(i, j int) bool {
		return params.Tables[i].NamePreffix < params.Tables[j].NamePreffix
	})

	s.ConstantsParams[outputDir] = params

	return nil
}
