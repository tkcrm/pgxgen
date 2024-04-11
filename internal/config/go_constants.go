package config

type GoConstants struct {
	Tables GoConstantsTables `yaml:"tables"`
}

type GoConstantsTables map[string]GoConstantsTablesItem

type GoConstantsTablesItem struct {
	OutputDir          string `yaml:"output_dir"`
	IncludeColumnNames bool   `yaml:"include_column_names"`
}
