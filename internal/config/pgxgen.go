package config

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type Pgxgen struct {
	Version                  string                     `yaml:"version"`
	Sqlc                     []PgxgenSqlc               `yaml:"sqlc"`
	GenModels                []GenModels                `yaml:"gen_models"`
	GenKeystoneFromStruct    []GenKeystoneFromStruct    `yaml:"gen_keystone_models"`
	GenTypescriptFromStructs []GenTypescriptFromStructs `yaml:"gen_typescript_from_structs"`
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
	Move                     SqlcModelsMove `yaml:"move"`
}

type SqlcModelsMove struct {
	OutputDir      string `yaml:"output_dir"`
	OutputFileName string `yaml:"output_file_name"`
	PackageName    string `yaml:"package_name"`
	PackagePath    string `yaml:"package_path"`
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
