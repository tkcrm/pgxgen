package config

import (
	"strings"

	"github.com/tkcrm/pgxgen/utils"
)

type Pgxgen struct {
	Version               int         `yaml:"version"`
	OutputCrudSqlFileName string      `yaml:"output_crud_sql_file_name"`
	GenModels             []GenModels `yaml:"gen_models"`
	JsonTags              JsonTags    `yaml:"json_tags"`
	CrudParams            CrudParams  `yaml:"crud_params"`
}

type GenModels struct {
	DeleteSqlcData              bool                          `yaml:"delete_sqlc_data"`
	ModelsOutputDir             string                        `yaml:"models_output_dir"`
	ModelsOutputFilename        string                        `yaml:"models_output_filename"`
	ModelsPackageName           string                        `yaml:"models_package_name"`
	ModelsImports               []string                      `yaml:"models_imports"`
	PrefereUintForIds           bool                          `yaml:"prefere_uint_for_ids"`
	PrefereUintForIdsExceptions []PrefereUintForIdsExceptions `yaml:"prefere_uint_for_ids_exceptions"`
	AddFields                   []AddFields                   `yaml:"add_fields"`
	UpdateFields                []UpdateFields                `yaml:"update_fields"`
	DeleteFields                []DeleteFields                `yaml:"delete_fields"`
}

type PrefereUintForIdsExceptions struct {
	StructName string   `yaml:"struct_name"`
	FieldNames []string `yaml:"field_names"`
}

func (s *GenModels) ExistPrefereExceptionsField(st_name, field_name string) bool {
	for _, item := range s.PrefereUintForIdsExceptions {
		if item.StructName == st_name && utils.ExistInStringArray(item.FieldNames, field_name) {
			return true
		}
	}
	return false
}

type AddFields struct {
	StructName string `yaml:"struct_name"`
	FieldName  string `yaml:"field_name"`
	Position   string `yaml:"position"`
	Type       string `yaml:"type"`
	Tags       []Tag  `yaml:"tags"`
}

type UpdateFields struct {
	StructName    string             `yaml:"struct_name"`
	FieldName     string             `yaml:"field_name"`
	NewParameters NewFieldParameters `yaml:"new_parameters"`
}

type DeleteFields struct {
	StructName string   `yaml:"struct_name"`
	FieldNames []string `yaml:"field_names"`
}

type NewFieldParameters struct {
	Name                 string `yaml:"name"`
	Type                 string `yaml:"type"`
	MatchWithCurrentTags bool   `yaml:"match_with_current_tags"`
	Tags                 []Tag  `yaml:"tags"`
}

type Tag struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type JsonTags struct {
	Omitempty []string `yaml:"omitempty"`
	Hide      []string `yaml:"hide"`
}

type CrudParams struct {
	Limit   []string  `yaml:"limit"`
	OrderBy []OrderBy `yaml:"order_by"`
	Where   []Where   `yaml:"where"`
}

// OrderBy used only for `Find` method
type OrderBy struct {
	By     string   `yaml:"by"`
	Order  string   `yaml:"order"`
	Tables []string `yaml:"tables"`
}

// `Where` used for all method except `create`. Instead of listing all methods, you can use an asterisk: *
type Where struct {
	Methods string   `yaml:"methods"`
	Tables  []string `yaml:"tables"`
	Params  []string `yaml:"params"`
}

func (p *Pgxgen) GetWhereParams(table, method string) []string {
	params := []string{}

	for _, item := range p.CrudParams.Where {
		method = strings.ToLower(method)
		if method == "c" || (item.Methods != "*" && !strings.Contains(strings.ToLower(item.Methods), method)) {
			continue
		}
		if !utils.ExistInStringArray(item.Tables, "*") && !utils.ExistInStringArray(item.Tables, table) {
			continue
		}
		for _, param := range item.Params {
			if utils.ExistInStringArray(params, param) {
				continue
			}
			params = append(params, param)
		}
	}

	return params
}

func (p *Pgxgen) GetOrderByParams(table string) *OrderBy {

	for _, item := range p.CrudParams.OrderBy {
		if !utils.ExistInStringArray(item.Tables, "*") && !utils.ExistInStringArray(item.Tables, table) {
			continue
		}
		if item.By == "" {
			continue
		}
		if item.Order == "" {
			item.Order = "DESC"
		}
		return &item
	}

	return nil
}

func (p *Pgxgen) GetLimitParam(table string) bool {
	if utils.ExistInStringArray(p.CrudParams.Limit, "*") {
		return true
	}
	return utils.ExistInStringArray(p.CrudParams.Limit, table)
}
