package goconstatnts

import (
	"regexp"
	"sort"

	"github.com/gobeam/stringy"
	"github.com/pkg/errors"
	cmnutils "github.com/tkcrm/modules/pkg/utils"
	"github.com/tkcrm/pgxgen/utils"
)

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
