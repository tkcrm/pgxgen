package constants

import (
	"bytes"
	"fmt"
	"go/format"
	"path/filepath"
	"sort"

	"github.com/gobeam/stringy"
	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/sqlparser/catalog"
	"github.com/tkcrm/pgxgen/pkg/logger"
	"github.com/tkcrm/pgxgen/utils"
)

const defaultFileName = "constants_gen.go"

// GenerateFromV2ConfigWithCatalog generates Go constants using a pre-parsed catalog.
func GenerateFromV2ConfigWithCatalog(
	l logger.Logger,
	configDir string,
	schema *config.SchemaConfig,
	cat *catalog.Catalog,
) error {
	return generateConstants(l, configDir, schema, cat)
}

func generateConstants(
	l logger.Logger,
	configDir string,
	schema *config.SchemaConfig,
	cat *catalog.Catalog,
) error {
	if len(schema.Tables) == 0 {
		return nil
	}

	defaultIncludeColumns := false
	if schema.Defaults != nil && schema.Defaults.Constants != nil {
		defaultIncludeColumns = schema.Defaults.Constants.IncludeColumnNames
	}

	type tableEntry struct {
		tableName      string
		includeColumns bool
	}
	grouped := make(map[string][]tableEntry)

	tableNames := make([]string, 0, len(schema.Tables))
	for name := range schema.Tables {
		tableNames = append(tableNames, name)
	}
	sort.Strings(tableNames)

	for _, tableName := range tableNames {
		tableConfig := schema.Tables[tableName]

		includeColumns := defaultIncludeColumns
		hasConstants := tableConfig.Constants != nil

		if hasConstants && tableConfig.Constants.IncludeColumnNames != nil {
			includeColumns = *tableConfig.Constants.IncludeColumnNames
		} else if !hasConstants && !defaultIncludeColumns {
			continue
		}

		outputDir := schema.ResolveOutputDir(tableName)
		if tableConfig.OutputDir != "" {
			outputDir = tableConfig.OutputDir
		}
		if outputDir == "" {
			continue
		}

		if !filepath.IsAbs(outputDir) {
			outputDir = filepath.Join(configDir, outputDir)
		}

		absDir, err := filepath.Abs(outputDir)
		if err != nil {
			return fmt.Errorf("abs path: %w", err)
		}

		grouped[absDir] = append(grouped[absDir], tableEntry{
			tableName:      tableName,
			includeColumns: includeColumns,
		})
	}

	if len(grouped) == 0 {
		return nil
	}

	for outputDir, entries := range grouped {
		packageName, err := utils.GetGoPackageNameForDir(outputDir)
		if err != nil || packageName == "" {
			packageName = filepath.Base(outputDir)
		}

		params := ConstantsParams{
			Package: packageName,
		}

		for _, entry := range entries {
			tablePreffix := stringy.New(entry.tableName).CamelCase().UcFirst()
			params.Tables = append(params.Tables, ConstantsTableNamesParamsItem{
				NamePreffix: tablePreffix,
				Name:        entry.tableName,
			})

			if entry.includeColumns {
				columns := getTableColumns(cat, entry.tableName)
				for _, col := range columns {
					colPreffix := tablePreffix + stringy.New(col).CamelCase().UcFirst()
					params.ColumnNames = append(params.ColumnNames, ConstantsColumnNamesParamsItem{
						TableName:   entry.tableName,
						NamePreffix: colPreffix,
						Name:        col,
					})
				}
			}
		}

		buf := new(bytes.Buffer)
		if err := RenderConstants(params, buf); err != nil {
			return fmt.Errorf("render constants: %w", err)
		}

		compiled, err := format.Source(buf.Bytes())
		if err != nil {
			// If format fails, use raw output
			compiled = buf.Bytes()
		}

		if err := utils.SaveFile(outputDir, defaultFileName, compiled); err != nil {
			return fmt.Errorf("save file: %w", err)
		}

		l.Infof("constants generated in %s", outputDir)
	}

	return nil
}

func getTableColumns(cat *catalog.Catalog, tableName string) []string {
	for _, s := range cat.Schemas {
		for _, t := range s.Tables {
			if t.Name == tableName {
				cols := make([]string, len(t.Columns))
				for i, c := range t.Columns {
					cols[i] = c.Name
				}
				return cols
			}
		}
	}
	return nil
}
