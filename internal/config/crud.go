package config

type MethodType string

func (m MethodType) String() string {
	return string(m)
}

type CrudParams struct {
	AutoRemoveGeneratedFiles    bool          `yaml:"auto_remove_generated_files"`
	ExcludeTableNameFromMethods bool          `yaml:"exclude_table_name_from_methods"`
	Default                     DefaultParams `yaml:"default"`
	Tables                      Table         `yaml:"tables"`
}

type DefaultParams struct {
	Methods map[MethodType]Method `yaml:"methods"`
}

type Table map[string]TableParams

type TableParams struct {
	PrimaryColumn string                `yaml:"primary_column"`
	OutputDir     string                `yaml:"output_dir"`
	Methods       map[MethodType]Method `yaml:"methods"`
}

type Method struct {
	Name            string                     `yaml:"name"`
	Returning       string                     `yaml:"returning"`
	Where           map[string]WhereParamsItem `yaml:"where"`
	WhereAdditional []string                   `yaml:"where_additional"`
	SkipColumns     []string                   `yaml:"skip_columns"`
	ColumnValues    map[string]string          `yaml:"column_values"`

	// For find method
	Limit bool       `yaml:"limit"`
	Order OrderParam `yaml:"order"`
}

type OrderParam struct {
	By        string `yaml:"by"`
	Direction string `yaml:"direction"`
}

type WhereParamsItem struct {
	Value string `yaml:"value"`
	// Default is = (equal)
	Operator string `yaml:"operator"`
}

func (s *Method) AddWhereParam(key string, params WhereParamsItem) {
	if s.Where == nil {
		s.Where = make(map[string]WhereParamsItem)
	}

	s.Where[key] = params
}
