package config

import (
	"strings"

	"github.com/tkcrm/pgxgen/utils"
)

type Config struct {
	Sqlc   SqlcConfig
	Pgxgen PgxgenConfig
}

type SqlcConfig struct {
	Version  int       `yaml:"version"`
	Packages []Package `yaml:"packages"`
}

type Package struct {
	Name                      string `yaml:"name"`
	Path                      string `yaml:"path"`
	Queries                   string `yaml:"queries"`
	Schema                    string `yaml:"schema"`
	Engine                    string `yaml:"engine"`
	EmitPreparedQueries       bool   `yaml:"emit_prepared_queries"`
	EmitInterface             bool   `yaml:"emit_interface"`
	EmitExactTableNames       bool   `yaml:"emit_exact_table_names"`
	EmitEmptySlices           bool   `yaml:"emit_empty_slices"`
	EmitExportedQueries       bool   `yaml:"emit_exported_queries"`
	EmitJsonTags              bool   `yaml:"emit_json_tags"`
	EmitResultStructPointers  bool   `yaml:"emit_result_struct_pointers"`
	EmitParamsStructPointers  bool   `yaml:"emit_params_struct_pointers"`
	EmitMethodsWithDbArgument bool   `yaml:"emit_methods_with_db_argument"`
	JsonTagsCaseStyle         string `yaml:"json_tags_case_style"`
	OutputDbFileName          string `yaml:"output_db_file_name"`
	OutputModelsFileName      string `yaml:"output_models_file_name"`
	OutputQuerierFileName     string `yaml:"output_querier_file_name"`
}

type PgxgenConfig struct {
	Version               int        `yaml:"version"`
	OutputCrudSqlFileName string     `yaml:"output_crud_sql_file_name"`
	JsonTags              JsonTags   `yaml:"json_tags"`
	CrudParams            CrudParams `yaml:"crud_params"`
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

func (p *PgxgenConfig) GetWhereParams(table, method string) []string {
	params := []string{}

	for _, item := range p.CrudParams.Where {
		method = strings.ToLower(method)
		if method == "c" || (item.Methods != "*" && !strings.Contains(strings.ToLower(item.Methods), method)) {
			continue
		}
		if !utils.ExistInArray(item.Tables, "*") && !utils.ExistInArray(item.Tables, table) {
			continue
		}
		for _, param := range item.Params {
			if utils.ExistInArray(params, param) {
				continue
			}
			params = append(params, param)
		}
	}

	return params
}

func (p *PgxgenConfig) GetOrderByParams(table string) *OrderBy {

	for _, item := range p.CrudParams.OrderBy {
		if !utils.ExistInArray(item.Tables, "*") && !utils.ExistInArray(item.Tables, table) {
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

func (p *PgxgenConfig) GetLimitParam(table string) bool {
	if utils.ExistInArray(p.CrudParams.Limit, "*") {
		return true
	}
	return utils.ExistInArray(p.CrudParams.Limit, table)
}
