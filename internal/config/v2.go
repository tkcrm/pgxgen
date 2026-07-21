package config

import "strings"

// V2Config is the root configuration for pgxgen v2.
type V2Config struct {
	Version   string           `yaml:"version" validate:"required,eq=2"`
	Schemas   []SchemaConfig   `yaml:"schemas" validate:"required,min=1,dive"`
	Templates *TemplatesConfig `yaml:"templates,omitempty"`
}

// SchemaConfig defines a single schema source with its engine, paths, and generation options.
type SchemaConfig struct {
	Name          string                 `yaml:"name" validate:"required"`
	Engine        string                 `yaml:"engine" validate:"required,oneof=postgresql mysql sqlite"`
	SchemaDir     string                 `yaml:"schema_dir" validate:"required"`
	Models        *ModelsConfig          `yaml:"models,omitempty"`
	Sqlc          *SqlcConfig            `yaml:"sqlc,omitempty"`
	Defaults      *DefaultsConfig        `yaml:"defaults,omitempty"`
	Tables        map[string]TableConfig `yaml:"tables,omitempty"`
	CustomQueries []CustomQueryConfig    `yaml:"custom_queries,omitempty"`
}

// ModelsConfig defines Go model generation settings.
type ModelsConfig struct {
	OutputDir             string               `yaml:"output_dir" validate:"required"`
	OutputFileName        string               `yaml:"output_file_name,omitempty"`
	PackageName           string               `yaml:"package_name" validate:"required"`
	PackagePath           string               `yaml:"package_path,omitempty"`
	CustomTypes           []string             `yaml:"custom_types,omitempty"`
	SkipTables            []string             `yaml:"skip_tables,omitempty"`
	SkipEnums             []string             `yaml:"skip_enums,omitempty"`
	EmitJsonTags          bool                 `yaml:"emit_json_tags,omitempty"`
	EmitDbTags            bool                 `yaml:"emit_db_tags,omitempty"`
	EmitPointersForNull   bool                 `yaml:"emit_pointers_for_null,omitempty"`
	IncludeStructComments bool                 `yaml:"include_struct_comments,omitempty"`
	SqlPackage            string               `yaml:"sql_package,omitempty" validate:"omitempty,oneof=pgx/v5 pgx/v4 database/sql"`
	TypeOverrides         []TypeOverrideConfig `yaml:"type_overrides,omitempty"`
	CustomTags            []CustomTagConfig    `yaml:"custom_tags,omitempty"`
}

// TypeOverrideConfig overrides SQL type → Go type mapping.
type TypeOverrideConfig struct {
	SqlType string `yaml:"sql_type" validate:"required"`
	GoType  string `yaml:"go_type" validate:"required"`
	Import  string `yaml:"import,omitempty"`
}

// CustomTagConfig defines an additional struct tag to emit.
type CustomTagConfig struct {
	Name   string `yaml:"name" validate:"required"`
	Format string `yaml:"format,omitempty"` // applied to NOT NULL columns
}

// GetOutputFileName returns the output file name, defaulting to "models.go".
func (c *ModelsConfig) GetOutputFileName() string {
	if c.OutputFileName != "" {
		return c.OutputFileName
	}
	return "models.go"
}

// SqlcConfig defines sqlc auto-generation settings.
type SqlcConfig struct {
	Defaults            *SqlcDefaultsConfig  `yaml:"defaults,omitempty"`
	Overrides           *SqlcOverridesConfig `yaml:"overrides,omitempty"`
	KeepGeneratedConfig bool                 `yaml:"keep_generated_config,omitempty"`
}

// SqlcDefaultsConfig holds default sqlc gen.go options applied to all repos.
type SqlcDefaultsConfig struct {
	SqlPackage               string `yaml:"sql_package,omitempty" validate:"omitempty,oneof=pgx/v5 pgx/v4 database/sql"`
	EmitPreparedQueries      bool   `yaml:"emit_prepared_queries"`
	EmitInterface            bool   `yaml:"emit_interface"`
	EmitJsonTags             bool   `yaml:"emit_json_tags"`
	EmitDbTags               bool   `yaml:"emit_db_tags"`
	EmitExportedQueries      bool   `yaml:"emit_exported_queries"`
	EmitExactTableNames      bool   `yaml:"emit_exact_table_names"`
	EmitEmptySlices          bool   `yaml:"emit_empty_slices"`
	EmitResultStructPointers bool   `yaml:"emit_result_struct_pointers"`
	EmitParamsStructPointers bool   `yaml:"emit_params_struct_pointers"`
	EmitEnumValidMethod      bool   `yaml:"emit_enum_valid_method"`
	EmitAllEnumValues        bool   `yaml:"emit_all_enum_values"`
	QueryParameterLimit      *int   `yaml:"query_parameter_limit,omitempty"`
	JsonTagsCaseStyle        string `yaml:"json_tags_case_style,omitempty"`
}

// SqlcOverridesConfig maps 1:1 to sqlc's overrides section.
type SqlcOverridesConfig struct {
	Rename  map[string]string    `yaml:"rename,omitempty"`
	Types   []SqlcTypeOverride   `yaml:"types,omitempty"`
	Columns []SqlcColumnOverride `yaml:"columns,omitempty"`
}

// SqlcTypeOverride defines a db_type → go_type mapping.
type SqlcTypeOverride struct {
	DbType   string `yaml:"db_type" validate:"required"`
	GoType   any    `yaml:"go_type" validate:"required"` // string or {type: ..., import: ...}
	Nullable bool   `yaml:"nullable,omitempty"`
}

// SqlcColumnOverride defines per-column overrides (struct tags, go types).
type SqlcColumnOverride struct {
	Column      string `yaml:"column" validate:"required"`
	GoType      any    `yaml:"go_type,omitempty"`
	GoStructTag string `yaml:"go_struct_tag,omitempty"`
}

// DefaultsConfig defines defaults inherited by all tables in this schema.
type DefaultsConfig struct {
	// Pattern A: per-table repos
	QueriesDirPrefix string `yaml:"queries_dir_prefix,omitempty"`
	OutputDirPrefix  string `yaml:"output_dir_prefix,omitempty"`

	// Package naming for per-table repos
	PackagePrefix string `yaml:"package_prefix,omitempty"`
	PackageSuffix string `yaml:"package_suffix,omitempty"`

	// Pattern B: single repo
	QueriesDir string `yaml:"queries_dir,omitempty"`
	OutputDir  string `yaml:"output_dir,omitempty"`

	Crud      *CrudDefaultsConfig      `yaml:"crud,omitempty"`
	Constants *ConstantsDefaultsConfig `yaml:"constants,omitempty"`
}

// resolvePackageName returns the table name with optional prefix and suffix applied.
func (d *DefaultsConfig) resolvePackageName(tableName string) string {
	return d.PackagePrefix + tableName + d.PackageSuffix
}

// CrudDefaultsConfig defines default CRUD settings for all tables.
type CrudDefaultsConfig struct {
	AutoClean        bool                     `yaml:"auto_clean,omitempty"`
	ExcludeTableName bool                     `yaml:"exclude_table_name,omitempty"`
	Methods          map[string]*MethodConfig `yaml:"methods,omitempty"`
}

// ConstantsDefaultsConfig defines default constants settings.
type ConstantsDefaultsConfig struct {
	IncludeColumnNames bool `yaml:"include_column_names,omitempty"`
}

// TableConfig defines per-table settings (crud + constants + sqlc overrides).
type TableConfig struct {
	PrimaryColumn string                `yaml:"primary_column,omitempty"`
	QueriesDir    string                `yaml:"queries_dir,omitempty"` // override
	OutputDir     string                `yaml:"output_dir,omitempty"`  // override
	Sqlc          *TableSqlcConfig      `yaml:"sqlc,omitempty"`
	Crud          *TableCrudConfig      `yaml:"crud,omitempty"`
	Constants     *TableConstantsConfig `yaml:"constants,omitempty"`
	SoftDelete    *SoftDeleteConfig     `yaml:"soft_delete,omitempty"`
}

// TableSqlcConfig holds per-table sqlc overrides.
type TableSqlcConfig struct {
	QueryParameterLimit *int `yaml:"query_parameter_limit,omitempty"`
}

// TableCrudConfig holds per-table CRUD settings.
type TableCrudConfig struct {
	Methods map[string]*MethodConfig `yaml:"methods,omitempty"`
}

// TableConstantsConfig holds per-table constants settings.
type TableConstantsConfig struct {
	IncludeColumnNames *bool `yaml:"include_column_names,omitempty"`
}

// MethodConfig defines a single CRUD method.
type MethodConfig struct {
	Name            string                      `yaml:"name,omitempty"`
	Returning       string                      `yaml:"returning,omitempty"`
	Where           map[string]WhereParamConfig `yaml:"where,omitempty"`
	WhereAdditional []string                    `yaml:"where_additional,omitempty"`
	SkipColumns     []string                    `yaml:"skip_columns,omitempty"`
	ColumnValues    map[string]string           `yaml:"column_values,omitempty"`
	Limit           bool                        `yaml:"limit,omitempty"`
	Order           *OrderConfig                `yaml:"order,omitempty"`
}

// WhereParamConfig defines a WHERE condition.
type WhereParamConfig struct {
	Value    string `yaml:"value,omitempty"`
	Operator string `yaml:"operator,omitempty"` // default: =
}

// OrderConfig defines ORDER BY settings.
type OrderConfig struct {
	By        string `yaml:"by" validate:"required"`
	Direction string `yaml:"direction,omitempty"` // default: DESC
}

// GetDirection returns the direction, defaulting to DESC.
func (o *OrderConfig) GetDirection() string {
	if o.Direction != "" {
		return o.Direction
	}
	return "DESC"
}

// SoftDeleteConfig enables soft delete for a table.
type SoftDeleteConfig struct {
	Column string `yaml:"column" validate:"required"`
}

// CustomQueryConfig defines a custom SQL query.
type CustomQueryConfig struct {
	Name      string `yaml:"name" validate:"required"`
	Type      string `yaml:"type" validate:"required,oneof=one many exec copyfrom"`
	Table     string `yaml:"table,omitempty"`
	OutputDir string `yaml:"output_dir,omitempty"`
	SQL       string `yaml:"sql" validate:"required"`
}

// TemplatesConfig allows user-provided templates.
type TemplatesConfig struct {
	CrudDir   string `yaml:"crud_dir,omitempty"`
	ModelsDir string `yaml:"models_dir,omitempty"`
}

// ResolveQueriesDir returns the queries directory for a table.
func (s *SchemaConfig) ResolveQueriesDir(tableName string) string {
	if s.Defaults != nil {
		if s.Defaults.QueriesDirPrefix != "" {
			return s.Defaults.QueriesDirPrefix + "/" + tableName
		}
		if s.Defaults.QueriesDir != "" {
			return s.Defaults.QueriesDir
		}
	}
	return ""
}

// ResolveOutputDir returns the sqlc output directory for a table.
func (s *SchemaConfig) ResolveOutputDir(tableName string) string {
	if s.Defaults != nil {
		if s.Defaults.OutputDirPrefix != "" {
			return s.Defaults.OutputDirPrefix + "/" + s.Defaults.resolvePackageName(tableName)
		}
		if s.Defaults.OutputDir != "" {
			return s.Defaults.OutputDir
		}
	}
	return ""
}

// IsPerTableMode returns true if per-table repos pattern is used.
func (s *SchemaConfig) IsPerTableMode() bool {
	return s.Defaults != nil && s.Defaults.QueriesDirPrefix != ""
}

// TableKeyParts holds the parsed components of a table key from pgxgen.yaml.
// A key like "shop.orders" is split into Schema="shop", Table="orders".
// A key like "orders" is split into Schema="", Table="orders".
type TableKeyParts struct {
	Schema string // database schema name, empty for default/public
	Table  string // bare table name without schema prefix
}

// SQLTableName returns the table name as it should appear in SQL statements.
// If a schema is specified, it returns "schema.table"; otherwise just "table".
func (p TableKeyParts) SQLTableName() string {
	if p.Schema != "" {
		return p.Schema + "." + p.Table
	}
	return p.Table
}

// GoName returns the name used for Go identifiers, file paths, and directories.
// For schema-qualified keys, it prepends the schema name with an underscore
// (e.g. "shop_orders") to avoid collisions between same-named tables in
// different schemas. For plain table keys, it returns the bare table name.
func (p TableKeyParts) GoName() string {
	if p.Schema != "" {
		return p.Schema + "_" + p.Table
	}
	return p.Table
}

// ParseTableKey splits a pgxgen.yaml table map key into schema and table parts.
// Supports "schema.table" (dot-separated) and plain "table" formats.
func ParseTableKey(key string) TableKeyParts {
	if idx := strings.LastIndex(key, "."); idx != -1 {
		return TableKeyParts{
			Schema: key[:idx],
			Table:  key[idx+1:],
		}
	}
	return TableKeyParts{Table: key}
}
