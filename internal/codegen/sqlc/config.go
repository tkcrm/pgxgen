package sqlc

import (
	"path/filepath"
	"sort"

	"github.com/tkcrm/pgxgen/internal/config"
)

// sqlcConfig represents the sqlc.yaml v2 format.
type sqlcConfig struct {
	Version   string         `yaml:"version"`
	Overrides *sqlcOverrides `yaml:"overrides,omitempty"`
	SQL       []sqlcSQLEntry `yaml:"sql"`
}

type sqlcOverrides struct {
	Go *sqlcGoOverrides `yaml:"go,omitempty"`
}

type sqlcGoOverrides struct {
	Rename    map[string]string `yaml:"rename,omitempty"`
	Overrides []interface{}     `yaml:"overrides,omitempty"`
}

type sqlcSQLEntry struct {
	Schema  string  `yaml:"schema"`
	Queries string  `yaml:"queries"`
	Engine  string  `yaml:"engine"`
	Gen     sqlcGen `yaml:"gen"`
}

type sqlcGen struct {
	Go sqlcGenGo `yaml:"go"`
}

type sqlcGenGo struct {
	Out                      string `yaml:"out"`
	Package                  string `yaml:"package,omitempty"`
	SqlPackage               string `yaml:"sql_package,omitempty"`
	EmitPreparedQueries      bool   `yaml:"emit_prepared_queries"`
	EmitInterface            bool   `yaml:"emit_interface"`
	EmitJsonTags             bool   `yaml:"emit_json_tags"`
	EmitExportedQueries      bool   `yaml:"emit_exported_queries"`
	EmitDbTags               bool   `yaml:"emit_db_tags"`
	EmitExactTableNames      bool   `yaml:"emit_exact_table_names"`
	EmitEmptySlices          bool   `yaml:"emit_empty_slices"`
	EmitResultStructPointers bool   `yaml:"emit_result_struct_pointers"`
	EmitParamsStructPointers bool   `yaml:"emit_params_struct_pointers"`
	EmitEnumValidMethod      bool   `yaml:"emit_enum_valid_method"`
	EmitAllEnumValues        bool   `yaml:"emit_all_enum_values"`
	QueryParameterLimit      *int   `yaml:"query_parameter_limit,omitempty"`
	JsonTagsCaseStyle        string `yaml:"json_tags_case_style,omitempty"`
}

// BuildSqlcConfig constructs a sqlc.yaml config from a pgxgen v2 schema.
// All paths are prefixed with "../" because the generated sqlc.yaml lives in
// .pgxgen/sqlc.yaml (one level deeper than the project root).
func BuildSqlcConfig(schema *config.SchemaConfig) *sqlcConfig {
	cfg := &sqlcConfig{
		Version: "2",
	}

	// Map overrides
	if schema.Sqlc != nil && schema.Sqlc.Overrides != nil {
		overrides := schema.Sqlc.Overrides
		goOverrides := &sqlcGoOverrides{}

		if overrides.Rename != nil {
			goOverrides.Rename = overrides.Rename
		}

		// Combine type overrides and column overrides into sqlc format
		var allOverrides []interface{}
		for _, t := range overrides.Types {
			entry := map[string]interface{}{
				"db_type": t.DbType,
				"go_type": t.GoType,
			}
			if t.Nullable {
				entry["nullable"] = true
			}
			allOverrides = append(allOverrides, entry)
		}
		for _, c := range overrides.Columns {
			entry := map[string]interface{}{
				"column": c.Column,
			}
			if c.GoType != nil {
				entry["go_type"] = c.GoType
			}
			if c.GoStructTag != "" {
				entry["go_struct_tag"] = c.GoStructTag
			}
			allOverrides = append(allOverrides, entry)
		}

		if len(allOverrides) > 0 {
			goOverrides.Overrides = allOverrides
		}

		if goOverrides.Rename != nil || len(goOverrides.Overrides) > 0 {
			cfg.Overrides = &sqlcOverrides{Go: goOverrides}
		}
	}

	// Build SQL entries from tables
	defaults := getDefaults(schema)

	// Sort table names for deterministic output
	tableNames := make([]string, 0, len(schema.Tables))
	for name := range schema.Tables {
		tableNames = append(tableNames, name)
	}
	sort.Strings(tableNames)

	for _, tableName := range tableNames {
		tableConfig := schema.Tables[tableName]

		queriesDir := schema.ResolveQueriesDir(tableName)
		if tableConfig.QueriesDir != "" {
			queriesDir = tableConfig.QueriesDir
		}

		outputDir := schema.ResolveOutputDir(tableName)
		if tableConfig.OutputDir != "" {
			outputDir = tableConfig.OutputDir
		}

		if queriesDir == "" || outputDir == "" {
			continue
		}

		entry := sqlcSQLEntry{
			Schema:  relFromPgxgenDir(schema.SchemaDir),
			Queries: relFromPgxgenDir(queriesDir),
			Engine:  schema.Engine,
			Gen: sqlcGen{
				Go: sqlcGenGo{
					Out:                      relFromPgxgenDir(outputDir),
					SqlPackage:               defaults.SqlPackage,
					EmitInterface:            defaults.EmitInterface,
					EmitJsonTags:             defaults.EmitJsonTags,
					EmitDbTags:               defaults.EmitDbTags,
					EmitEmptySlices:          defaults.EmitEmptySlices,
					EmitResultStructPointers: defaults.EmitResultStructPointers,
					EmitEnumValidMethod:      defaults.EmitEnumValidMethod,
					EmitAllEnumValues:        defaults.EmitAllEnumValues,
					EmitPreparedQueries:      defaults.EmitPreparedQueries,
					EmitExportedQueries:      defaults.EmitExportedQueries,
					EmitExactTableNames:      defaults.EmitExactTableNames,
					EmitParamsStructPointers: defaults.EmitParamsStructPointers,
					QueryParameterLimit:      defaults.QueryParameterLimit,
					JsonTagsCaseStyle:        defaults.JsonTagsCaseStyle,
				},
			},
		}

		// Apply per-table sqlc overrides
		if tableConfig.Sqlc != nil {
			if tableConfig.Sqlc.QueryParameterLimit != nil {
				entry.Gen.Go.QueryParameterLimit = tableConfig.Sqlc.QueryParameterLimit
			}
		}

		cfg.SQL = append(cfg.SQL, entry)
	}

	return cfg
}

func getDefaults(schema *config.SchemaConfig) config.SqlcDefaultsConfig {
	if schema.Sqlc != nil && schema.Sqlc.Defaults != nil {
		return *schema.Sqlc.Defaults
	}
	return config.SqlcDefaultsConfig{}
}

// relFromPgxgenDir prefixes a relative path with "../" so it resolves correctly
// from .pgxgen/sqlc.yaml back to the project root.
func relFromPgxgenDir(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join("..", p)
}
