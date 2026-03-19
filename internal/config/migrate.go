package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// --- v1 types (only used for migration, not exported) ---

type v1Pgxgen struct {
	Version  string       `yaml:"version"`
	Sqlc     []v1SqlcItem `yaml:"sqlc"`
	Generate struct {
		Models []v1GenerateModelsConfig `yaml:"models"`
	} `yaml:"generate"`
}

type v1SqlcItem struct {
	SchemaDir   string        `yaml:"schema_dir"`
	SqlcModels  v1SqlcModels  `yaml:"models"`
	CrudParams  v1CrudParams  `yaml:"crud"`
	GoConstants v1GoConstants `yaml:"constants"`
}

type v1SqlcModels struct {
	ReplaceSqlcNullableTypes bool             `yaml:"replace_sqlc_nullable_types"`
	IncludeStructComments    bool             `yaml:"include_struct_comments"`
	Move                     v1SqlcModelsMove `yaml:"move"`
}

type v1SqlcModelsMove struct {
	OutputDir      string                    `yaml:"output_dir"`
	OutputFileName string                    `yaml:"output_file_name"`
	PackageName    string                    `yaml:"package_name"`
	PackagePath    string                    `yaml:"package_path"`
	Imports        []v1SqlcModelsMoveImports `yaml:"imports"`
}

func (s v1SqlcModelsMove) isUsable() bool {
	return s.OutputDir != "" || s.OutputFileName != "" || s.PackageName != "" || s.PackagePath != ""
}

type v1SqlcModelsMoveImports struct {
	Path   string `yaml:"path"`
	GoType string `yaml:"go_type"`
}

type v1CrudParams struct {
	AutoRemoveGeneratedFiles    bool                     `yaml:"auto_remove_generated_files"`
	ExcludeTableNameFromMethods bool                     `yaml:"exclude_table_name_from_methods"`
	Default                     v1DefaultParams          `yaml:"default"`
	Tables                      map[string]v1TableParams `yaml:"tables"`
}

type v1DefaultParams struct {
	Methods map[string]v1Method `yaml:"methods"`
}

type v1TableParams struct {
	PrimaryColumn string              `yaml:"primary_column"`
	OutputDir     string              `yaml:"output_dir"`
	Methods       map[string]v1Method `yaml:"methods"`
}

type v1Method struct {
	Name            string                       `yaml:"name"`
	Returning       string                       `yaml:"returning"`
	Where           map[string]v1WhereParamsItem `yaml:"where"`
	WhereAdditional []string                     `yaml:"where_additional"`
	SkipColumns     []string                     `yaml:"skip_columns"`
	ColumnValues    map[string]string            `yaml:"column_values"`
	Limit           bool                         `yaml:"limit"`
	Order           v1OrderParam                 `yaml:"order"`
}

type v1OrderParam struct {
	By        string `yaml:"by"`
	Direction string `yaml:"direction"`
}

type v1WhereParamsItem struct {
	Value    string `yaml:"value"`
	Operator string `yaml:"operator"`
}

type v1GoConstants struct {
	Tables map[string]v1GoConstantsTablesItem `yaml:"tables"`
}

type v1GoConstantsTablesItem struct {
	OutputDir          string `yaml:"output_dir"`
	IncludeColumnNames bool   `yaml:"include_column_names"`
}

type v1GenerateModelsConfig struct {
	SchemaDir           string `yaml:"schema_dir"`
	Engine              string `yaml:"engine"`
	OutputDir           string `yaml:"output_dir"`
	OutputFileName      string `yaml:"output_file_name"`
	PackageName         string `yaml:"package_name"`
	SqlPackage          string `yaml:"sql_package"`
	EmitJsonTags        bool   `yaml:"emit_json_tags"`
	EmitDbTags          bool   `yaml:"emit_db_tags"`
	EmitPointersForNull bool   `yaml:"emit_pointers_for_null"`
}

type v1SqlcConfig struct {
	Version string `yaml:"version"`
	SQL     []struct {
		Schema  string `yaml:"schema"`
		Queries string `yaml:"queries"`
		Engine  string `yaml:"engine"`
		Gen     struct {
			Go struct {
				SqlPackage               string `yaml:"sql_package"`
				Out                      string `yaml:"out"`
				EmitInterface            bool   `yaml:"emit_interface"`
				EmitJsonTags             bool   `yaml:"emit_json_tags"`
				EmitDbTags               bool   `yaml:"emit_db_tags"`
				EmitEmptySlices          bool   `yaml:"emit_empty_slices"`
				EmitResultStructPointers bool   `yaml:"emit_result_struct_pointers"`
				EmitEnumValidMethod      bool   `yaml:"emit_enum_valid_method"`
				EmitAllEnumValues        bool   `yaml:"emit_all_enum_values"`
				QueryParameterLimit      *int   `yaml:"query_parameter_limit,omitempty"`
			} `yaml:"go"`
		} `yaml:"gen"`
	} `yaml:"sql"`
	Overrides *struct {
		Go *struct {
			Rename    map[string]string        `yaml:"rename,omitempty"`
			Overrides []map[string]interface{} `yaml:"overrides,omitempty"`
		} `yaml:"go,omitempty"`
	} `yaml:"overrides,omitempty"`
}

// --- Migration logic ---

// MigrateV1ToV2 reads v1 pgxgen.yaml and optionally sqlc.yaml, returns v2 YAML.
func MigrateV1ToV2(pgxgenPath, sqlcPath string) ([]byte, error) {
	pgxgenData, err := os.ReadFile(pgxgenPath)
	if err != nil {
		return nil, fmt.Errorf("read pgxgen config: %w", err)
	}

	var v1 v1Pgxgen
	if err := yaml.Unmarshal(pgxgenData, &v1); err != nil {
		return nil, fmt.Errorf("parse v1 pgxgen config: %w", err)
	}

	var sqlcCfg *v1SqlcConfig
	if sqlcPath != "" {
		if data, err := os.ReadFile(sqlcPath); err == nil {
			var sc v1SqlcConfig
			if yaml.Unmarshal(data, &sc) == nil {
				sqlcCfg = &sc
			}
		}
	}

	v2 := &V2Config{Version: "2"}

	for _, item := range v1.Sqlc {
		v2.Schemas = append(v2.Schemas, migrateSchemaFromV1(item, sqlcCfg))
	}

	for _, m := range v1.Generate.Models {
		engine := m.Engine
		if engine == "" {
			engine = "postgresql"
		}
		v2.Schemas = append(v2.Schemas, SchemaConfig{
			Name:      inferSchemaName(m.SchemaDir),
			Engine:    engine,
			SchemaDir: m.SchemaDir,
			Models: &ModelsConfig{
				OutputDir:           m.OutputDir,
				OutputFileName:      m.OutputFileName,
				PackageName:         m.PackageName,
				SqlPackage:          m.SqlPackage,
				EmitJsonTags:        m.EmitJsonTags,
				EmitDbTags:          m.EmitDbTags,
				EmitPointersForNull: m.EmitPointersForNull,
			},
		})
	}

	out, err := yaml.Marshal(v2)
	if err != nil {
		return nil, fmt.Errorf("marshal v2 config: %w", err)
	}
	return out, nil
}

func migrateSchemaFromV1(item v1SqlcItem, sqlcCfg *v1SqlcConfig) SchemaConfig {
	engine := "postgresql"
	if sqlcCfg != nil && len(sqlcCfg.SQL) > 0 {
		engine = sqlcCfg.SQL[0].Engine
	}

	schema := SchemaConfig{
		Name:      inferSchemaName(item.SchemaDir),
		Engine:    engine,
		SchemaDir: item.SchemaDir,
	}

	if item.SqlcModels.Move.isUsable() {
		customTypes := make([]string, 0, len(item.SqlcModels.Move.Imports))
		for _, imp := range item.SqlcModels.Move.Imports {
			if imp.GoType != "" {
				customTypes = append(customTypes, imp.GoType)
			}
		}
		schema.Models = &ModelsConfig{
			OutputDir:             item.SqlcModels.Move.OutputDir,
			OutputFileName:        item.SqlcModels.Move.OutputFileName,
			PackageName:           item.SqlcModels.Move.PackageName,
			PackagePath:           item.SqlcModels.Move.PackagePath,
			CustomTypes:           customTypes,
			IncludeStructComments: item.SqlcModels.IncludeStructComments,
		}
		// Copy emit_json_tags/emit_db_tags from sqlc defaults to models
		if sqlcCfg != nil && len(sqlcCfg.SQL) > 0 {
			first := sqlcCfg.SQL[0].Gen.Go
			schema.Models.EmitJsonTags = first.EmitJsonTags
			schema.Models.EmitDbTags = first.EmitDbTags
		}
	}

	if sqlcCfg != nil && len(sqlcCfg.SQL) > 0 {
		first := sqlcCfg.SQL[0].Gen.Go
		schema.Sqlc = &SqlcConfig{
			Defaults: &SqlcDefaultsConfig{
				SqlPackage:               first.SqlPackage,
				EmitInterface:            first.EmitInterface,
				EmitJsonTags:             first.EmitJsonTags,
				EmitDbTags:               first.EmitDbTags,
				EmitEmptySlices:          first.EmitEmptySlices,
				EmitResultStructPointers: first.EmitResultStructPointers,
				EmitEnumValidMethod:      first.EmitEnumValidMethod,
				EmitAllEnumValues:        first.EmitAllEnumValues,
			},
		}
		if sqlcCfg.Overrides != nil && sqlcCfg.Overrides.Go != nil {
			overrides := &SqlcOverridesConfig{}
			if sqlcCfg.Overrides.Go.Rename != nil {
				overrides.Rename = sqlcCfg.Overrides.Go.Rename
			}
			// Parse type and column overrides from the overrides array
			for _, entry := range sqlcCfg.Overrides.Go.Overrides {
				if col, ok := entry["column"]; ok {
					co := SqlcColumnOverride{Column: fmt.Sprintf("%v", col)}
					if v, ok := entry["go_type"]; ok {
						co.GoType = v
					}
					if v, ok := entry["go_struct_tag"]; ok {
						co.GoStructTag = fmt.Sprintf("%v", v)
					}
					overrides.Columns = append(overrides.Columns, co)
				} else if dbType, ok := entry["db_type"]; ok {
					to := SqlcTypeOverride{
						DbType: fmt.Sprintf("%v", dbType),
						GoType: entry["go_type"],
					}
					if v, ok := entry["nullable"]; ok {
						if b, ok := v.(bool); ok {
							to.Nullable = b
						}
					}
					overrides.Types = append(overrides.Types, to)
				}
			}
			schema.Sqlc.Overrides = overrides
		}
	}

	queriesDirPrefix, outputDirPrefix := detectPrefixes(item, sqlcCfg)
	schema.Defaults = &DefaultsConfig{
		QueriesDirPrefix: queriesDirPrefix,
		OutputDirPrefix:  outputDirPrefix,
		Crud: &CrudDefaultsConfig{
			AutoClean:        item.CrudParams.AutoRemoveGeneratedFiles,
			ExcludeTableName: item.CrudParams.ExcludeTableNameFromMethods,
		},
	}

	if item.CrudParams.Default.Methods != nil {
		schema.Defaults.Crud.Methods = migrateMethodsMap(item.CrudParams.Default.Methods)
	}

	// Build query_parameter_limit map from sqlc SQL entries (queries dir → limit)
	qplMap := make(map[string]*int)
	if sqlcCfg != nil {
		for _, sqlEntry := range sqlcCfg.SQL {
			if sqlEntry.Gen.Go.QueryParameterLimit != nil {
				qplMap[sqlEntry.Queries] = sqlEntry.Gen.Go.QueryParameterLimit
			}
		}
	}

	schema.Tables = make(map[string]TableConfig, len(item.CrudParams.Tables))
	for tableName, tableParams := range item.CrudParams.Tables {
		tc := TableConfig{PrimaryColumn: tableParams.PrimaryColumn}
		if tableParams.Methods != nil {
			tc.Crud = &TableCrudConfig{Methods: migrateMethodsMap(tableParams.Methods)}
		}
		// Check if this table's queries dir has a query_parameter_limit
		if tableParams.OutputDir != "" {
			if qpl, ok := qplMap[tableParams.OutputDir]; ok {
				tc.Sqlc = &TableSqlcConfig{QueryParameterLimit: qpl}
			}
		}
		schema.Tables[tableName] = tc
	}

	for tableName, constItem := range item.GoConstants.Tables {
		tc := schema.Tables[tableName]
		includeColumns := constItem.IncludeColumnNames
		tc.Constants = &TableConstantsConfig{IncludeColumnNames: &includeColumns}
		schema.Tables[tableName] = tc
	}

	return schema
}

func migrateMethodsMap(methods map[string]v1Method) map[string]*MethodConfig {
	result := make(map[string]*MethodConfig, len(methods))
	for methodType, method := range methods {
		mc := &MethodConfig{
			Name:            method.Name,
			Returning:       method.Returning,
			SkipColumns:     method.SkipColumns,
			ColumnValues:    method.ColumnValues,
			WhereAdditional: method.WhereAdditional,
			Limit:           method.Limit,
		}
		if method.Order.By != "" {
			mc.Order = &OrderConfig{By: method.Order.By, Direction: method.Order.Direction}
		}
		if len(method.Where) > 0 {
			mc.Where = make(map[string]WhereParamConfig, len(method.Where))
			for k, v := range method.Where {
				mc.Where[k] = WhereParamConfig{Value: v.Value, Operator: v.Operator}
			}
		}
		result[methodType] = mc
	}
	return result
}

func detectPrefixes(item v1SqlcItem, sqlcCfg *v1SqlcConfig) (string, string) {
	var queriesDirPrefix, outputDirPrefix string
	for _, tableParams := range item.CrudParams.Tables {
		if tableParams.OutputDir != "" {
			queriesDirPrefix = filepath.Dir(tableParams.OutputDir)
			break
		}
	}
	if sqlcCfg != nil && len(sqlcCfg.SQL) > 0 {
		if out := sqlcCfg.SQL[0].Gen.Go.Out; out != "" {
			outputDirPrefix = filepath.Dir(out)
		}
	}
	return queriesDirPrefix, outputDirPrefix
}

func inferSchemaName(schemaDir string) string {
	base := filepath.Base(schemaDir)
	if base == "." || base == "/" {
		return "main"
	}
	name := strings.ReplaceAll(base, "/", "-")
	if name == "migrations" || name == "sql" {
		return "main"
	}
	return name
}
