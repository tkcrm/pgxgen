package config

type Pgxgen struct {
	Version                             string                     `yaml:"version"`
	DisableAutoReaplceSqlcNullableTypes bool                       `yaml:"disable_auto_replace_sqlc_nullable_types"`
	SqlcModels                          SqlcMoveModels             `yaml:"sqlc_move_models"`
	GenModels                           []GenModels                `yaml:"gen_models"`
	GenKeystoneFromStruct               []GenKeystoneFromStruct    `yaml:"gen_keystone_models"`
	GenTypescriptFromStructs            []GenTypescriptFromStructs `yaml:"gen_typescript_from_structs"`
	JsonTags                            JsonTags                   `yaml:"json_tags"`
	CrudParams                          CrudParams                 `yaml:"crud_params"`
	GoConstants                         GoConstants                `yaml:"go_constants"`
}

type SqlcMoveModels struct {
	OutputDir      string `yaml:"output_dir"`
	OutputFilename string `yaml:"output_filename"`
	PackageName    string `yaml:"package_name"`
	PackagePath    string `yaml:"package_path"`
}

type JsonTags struct {
	Omitempty []string `yaml:"omitempty"`
	Hide      []string `yaml:"hide"`
}
