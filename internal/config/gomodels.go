package config

import (
	"strings"

	"github.com/tkcrm/modules/pkg/utils"
)

type GenModels struct {
	InputFilePath           string                    `yaml:"input_file_path"`
	InputDir                string                    `yaml:"input_dir"`
	DeleteOriginalFiles     bool                      `yaml:"delete_original_files"`
	OutputDir               string                    `yaml:"output_dir"`
	OutputFileName          string                    `yaml:"output_file_name"`
	PackageName             string                    `yaml:"package_name"`
	Imports                 []string                  `yaml:"imports"`
	UseUintForIds           bool                      `yaml:"use_uint_for_ids"`
	UseUintForIdsExceptions []UseUintForIdsExceptions `yaml:"use_uint_for_ids_exceptions"`
	AddFields               []AddFields               `yaml:"add_fields"`
	UpdateFields            []UpdateFields            `yaml:"update_fields"`
	UpdateAllStructFields   UpdateAllStructFields     `yaml:"update_all_struct_fields"`
	DeleteFields            []DeleteFields            `yaml:"delete_fields"`
	Rename                  map[string]string         `yaml:"rename"`
	ExcludeStructs          []ExcludeStructsItem      `yaml:"exclude_structs"`
	IncludeStructs          []IncludeStructsItem      `yaml:"include_structs"`
}

type UseUintForIdsExceptions struct {
	StructName string   `yaml:"struct_name"`
	FieldNames []string `yaml:"field_names"`
}

func (s *GenModels) GetModelsOutputDir() string {
	res := s.OutputDir

	if string(res[len(res)-1]) == "/" {
		res = res[:len(res)-1]
	}
	return res
}

func (s *GenModels) GetModelsOutputFileName() string {
	res := s.OutputFileName
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

type ExcludeStructsItem struct {
	StructName string `yaml:"struct_name"`
}

type IncludeStructsItem struct {
	StructName string `yaml:"struct_name"`
}
