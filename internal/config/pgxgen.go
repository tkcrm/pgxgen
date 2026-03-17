package config

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type Pgxgen struct {
	Version  string         `yaml:"version"`
	Sqlc     []PgxgenSqlc   `yaml:"sqlc"`
	Generate GenerateConfig `yaml:"generate"`
}

type GenerateConfig struct {
	Models []GenerateModelsConfig `yaml:"models"`
}

type GenerateModelsConfig struct {
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

func (c GenerateModelsConfig) GetOutputFileName() string {
	if c.OutputFileName != "" {
		return c.OutputFileName
	}
	return "models.go"
}

func (c GenerateModelsConfig) Validate() error {
	return validation.ValidateStruct(
		&c,
		validation.Field(&c.SchemaDir, validation.Required),
		validation.Field(&c.OutputDir, validation.Required),
		validation.Field(&c.PackageName, validation.Required),
	)
}

type PgxgenSqlc struct {
	SchemaDir   string      `yaml:"schema_dir"`
	SqlcModels  SqlcModels  `yaml:"models"`
	CrudParams  CrudParams  `yaml:"crud"`
	GoConstants GoConstants `yaml:"constants"`
}

func (s PgxgenSqlc) Validate() error {
	return validation.ValidateStruct(
		&s,
		validation.Field(&s.SchemaDir, validation.Required),
	)
}

type SqlcModels struct {
	ReplaceSqlcNullableTypes bool           `yaml:"replace_sqlc_nullable_types"`
	IncludeStructComments    bool           `yaml:"include_struct_comments"`
	Move                     SqlcModelsMove `yaml:"move"`
}

type SqlcModelsMoveImports struct {
	Path   string `yaml:"path"`
	GoType string `yaml:"go_type"`
}

type SqlcModelsMove struct {
	OutputDir      string                  `yaml:"output_dir"`
	OutputFileName string                  `yaml:"output_file_name"`
	PackageName    string                  `yaml:"package_name"`
	PackagePath    string                  `yaml:"package_path"`
	Imports        []SqlcModelsMoveImports `yaml:"imports"`
}

func (s SqlcModelsMove) Validate() error {
	return validation.ValidateStruct(
		&s,
		validation.Field(&s.OutputDir, validation.Required),
		validation.Field(&s.OutputFileName, validation.Required),
		validation.Field(&s.PackagePath, validation.Required),
	)
}

func (s SqlcModelsMove) IsUsable() bool {
	return s.OutputDir != "" ||
		s.OutputFileName != "" ||
		s.PackageName != "" ||
		s.PackagePath != ""
}
