package config

type GenTypescriptFromStructs struct {
	Path                     string   `yaml:"path"`
	OutputDir                string   `yaml:"output_dir"`
	OutputFileName           string   `yaml:"output_file_name"`
	PrettierCode             bool     `yaml:"prettier_code"`
	ExportTypePrefix         string   `yaml:"export_type_prefix"`
	ExportTypeSuffix         string   `yaml:"export_type_suffix"`
	IncludeStructNamesRegexp []string `yaml:"include_struct_names_regexp"`
	ExcludeStructNamesRegexp []string `yaml:"exclude_struct_names_regexp"`
}
