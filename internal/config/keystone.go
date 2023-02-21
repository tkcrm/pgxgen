package config

import validation "github.com/go-ozzo/ozzo-validation/v4"

type GenKeystoneFromStruct struct {
	DecoratorModelNamePrefix string           `yaml:"decorator_model_name_prefix"`
	InputFilePath            string           `yaml:"input_file_path" json:"input_file_path"`
	OutputDir                string           `yaml:"output_dir" json:"output_dir"`
	OutputFileName           string           `yaml:"output_file_name"`
	Sort                     string           `yaml:"sort"`
	WithSetter               bool             `yaml:"with_setter"`
	ExportModelSuffix        string           `yaml:"export_model_suffix"`
	PrettierCode             bool             `yaml:"prettier_code"`
	Params                   []KeystoneParams `yaml:"params"`
	SkipModels               []string         `yaml:"skip_models"`
}

func (s GenKeystoneFromStruct) Validate() error {
	return validation.ValidateStruct(&s,
		validation.Field(&s.InputFilePath, validation.Required),
		validation.Field(&s.OutputDir, validation.Required),
	)
}

type KeystoneParams struct {
	StructName  string `yaml:"struct_name"`
	FieldName   string `yaml:"field_name"`
	FieldParams []struct {
		WithSetter bool `yaml:"with_setter"`
	} `yaml:"field_params"`
}
