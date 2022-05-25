package config

import (
	"strings"

	"github.com/tkcrm/pgxgen/utils"
)

type Pgxgen struct {
	Version                  int                        `yaml:"version"`
	OutputCrudSqlFileName    string                     `yaml:"output_crud_sql_file_name"`
	GenModels                []GenModels                `yaml:"gen_models"`
	GenTypescriptFromStructs []GenTypescriptFromStructs `yaml:"gen_typescript_from_structs"`
	JsonTags                 JsonTags                   `yaml:"json_tags"`
	CrudParams               CrudParams                 `yaml:"crud_params"`
}

type GenModels struct {
	DeleteSqlcData          bool                      `yaml:"delete_sqlc_data"`
	ModelsOutputDir         string                    `yaml:"models_output_dir"`
	ModelsOutputFilename    string                    `yaml:"models_output_filename"`
	ModelsPackageName       string                    `yaml:"models_package_name"`
	ModelsImports           []string                  `yaml:"models_imports"`
	UseUintForIds           bool                      `yaml:"use_uint_for_ids"`
	UseUintForIdsExceptions []UseUintForIdsExceptions `yaml:"use_uint_for_ids_exceptions"`
	AddFields               []AddFields               `yaml:"add_fields"`
	UpdateFields            []UpdateFields            `yaml:"update_fields"`
	UpdateAllStructFields   UpdateAllStructFields     `yaml:"update_all_struct_fields"`
	DeleteFields            []DeleteFields            `yaml:"delete_fields"`
	ExternalModels          ExternalModels            `yaml:"external_models"`
}

type UseUintForIdsExceptions struct {
	StructName string   `yaml:"struct_name"`
	FieldNames []string `yaml:"field_names"`
}

func (s *GenModels) GetModelsOutputDir() string {
	res := s.ModelsOutputDir

	if string(res[len(res)-1]) == "/" {
		res = res[:len(res)-1]
	}
	return res
}

func (s *GenModels) GetModelsOutputFileName() string {
	res := s.ModelsOutputFilename
	if res == "" {
		res = "models.go"
	}
	if res != "" && !strings.HasSuffix(res, ".go") {
		res += ".go"
	}
	return res
}

func (s *GenModels) ExistPrefereExceptionsField(st_name, field_name string) bool {
	for _, item := range s.UseUintForIdsExceptions {
		if item.StructName == st_name && utils.ExistInArray(item.FieldNames, field_name) {
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

type UpdateAllStructFields struct {
	ByField []ByField `yaml:"by_field"`
	ByType  []ByType  `yaml:"by_type"`
}

type ByField struct {
	FieldName            string `yaml:"field_name"`
	NewFieldName         string `yaml:"new_field_name"`
	NewType              string `yaml:"new_type"`
	MatchWithCurrentTags bool   `yaml:"match_with_current_tags"`
	Tags                 []Tag  `yaml:"tags"`
}

type ByType struct {
	Type                 string `yaml:"type"`
	NewType              string `yaml:"new_type"`
	MatchWithCurrentTags bool   `yaml:"match_with_current_tags"`
	Tags                 []Tag  `yaml:"tags"`
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

type ExternalModels struct {
	Keystone struct {
		DecoratorModelNamePrefix string           `yaml:"decorator_model_name_prefix"`
		OutputDir                string           `yaml:"output_dir"`
		OutputFileName           string           `yaml:"output_file_name"`
		Sort                     string           `yaml:"sort"`
		WithSetter               bool             `yaml:"with_setter"`
		ExportModelSuffix        string           `yaml:"export_model_suffix"`
		PrettierCode             bool             `yaml:"prettier_code"`
		Params                   []KeystoneParams `yaml:"params"`
	} `yaml:"keystone"`
}

type KeystoneParams struct {
	StructName  string `yaml:"struct_name"`
	FieldName   string `yaml:"field_name"`
	FieldParams []struct {
		WithSetter bool `yaml:"with_setter"`
	} `yaml:"field_params"`
}

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
