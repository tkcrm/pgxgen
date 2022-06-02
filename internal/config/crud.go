package config

type MethodType string

func (m MethodType) String() string {
	return string(m)
}

type CrudParams struct {
	Default DefaultParams `yaml:"default"`
	Tables  Table         `yaml:"tables"`
}

type DefaultParams struct {
	Methods map[MethodType]Method `yaml:"methods"`
}

type Table map[string]TableParams

type TableParams struct {
	PrimaryColumn string                `yaml:"primary_column"`
	Methods       map[MethodType]Method `yaml:"methods"`
}

type Method struct {
	Name        string                     `yaml:"name"`
	Returning   string                     `yaml:"returning"`
	Where       map[string]WhereParamsItem `yaml:"where"`
	SkipColumns []string                   `yaml:"skip_columns"`

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
