package config

import (
	"path/filepath"

	"github.com/tkcrm/modules/pkg/utils"
)

type Sqlc struct {
	Version  int       `yaml:"version"`
	Packages []Package `yaml:"packages"`
	SQL      []SqlcSQL `yaml:"sql"`
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

type SqlcSQL struct {
	Schema  string `yaml:"schema"`
	Queries string `yaml:"queries"`
	Engine  string `yaml:"engine"`
	Gen     struct {
		Go struct {
			EmitInterface             bool              `json:"emit_interface" yaml:"emit_interface"`
			EmitJSONTags              bool              `json:"emit_json_tags" yaml:"emit_json_tags"`
			EmitDBTags                bool              `json:"emit_db_tags" yaml:"emit_db_tags"`
			EmitPreparedQueries       bool              `json:"emit_prepared_queries" yaml:"emit_prepared_queries"`
			EmitExactTableNames       bool              `json:"emit_exact_table_names,omitempty" yaml:"emit_exact_table_names"`
			EmitEmptySlices           bool              `json:"emit_empty_slices,omitempty" yaml:"emit_empty_slices"`
			EmitExportedQueries       bool              `json:"emit_exported_queries" yaml:"emit_exported_queries"`
			EmitResultStructPointers  bool              `json:"emit_result_struct_pointers" yaml:"emit_result_struct_pointers"`
			EmitParamsStructPointers  bool              `json:"emit_params_struct_pointers" yaml:"emit_params_struct_pointers"`
			EmitMethodsWithDBArgument bool              `json:"emit_methods_with_db_argument,omitempty" yaml:"emit_methods_with_db_argument"`
			EmitEnumValidMethod       bool              `json:"emit_enum_valid_method,omitempty" yaml:"emit_enum_valid_method"`
			EmitAllEnumValues         bool              `json:"emit_all_enum_values,omitempty" yaml:"emit_all_enum_values"`
			JSONTagsCaseStyle         string            `json:"json_tags_case_style,omitempty" yaml:"json_tags_case_style"`
			Package                   string            `json:"package" yaml:"package"`
			Out                       string            `json:"out" yaml:"out"`
			Rename                    map[string]string `json:"rename,omitempty" yaml:"rename"`
			SQLPackage                string            `json:"sql_package" yaml:"sql_package"`
			OutputDBFileName          string            `json:"output_db_file_name,omitempty" yaml:"output_db_file_name"`
			OutputModelsFileName      string            `json:"output_models_file_name,omitempty" yaml:"output_models_file_name"`
			OutputQuerierFileName     string            `json:"output_querier_file_name,omitempty" yaml:"output_querier_file_name"`
			OutputFilesSuffix         string            `json:"output_files_suffix,omitempty" yaml:"output_files_suffix"`
		} `yaml:"go"`
	} `yaml:"gen"`
}

type GetPathsResponse struct {
	ModelsPaths     []string
	QueriesPaths    []string
	OutPaths        []string
	MigrationsPaths []string
}

func (s *Sqlc) GetPaths() GetPathsResponse {
	res := GetPathsResponse{}

	// process sqlc version 1
	if s.Version == 1 {
		for _, p := range s.Packages {
			modelFileName := p.OutputModelsFileName
			if modelFileName == "" {
				modelFileName = "models.go"
			}

			res.ModelsPaths = utils.AppendIfNotExistInArray(
				res.ModelsPaths,
				filepath.Join(p.Path, modelFileName),
				func(i string) bool {
					return i == filepath.Join(p.Path, modelFileName)
				},
			)
			res.QueriesPaths = utils.AppendIfNotExistInArray(res.QueriesPaths, p.Queries,
				func(i string) bool {
					return i == p.Queries
				},
			)
			res.OutPaths = utils.AppendIfNotExistInArray(res.OutPaths, p.Path,
				func(i string) bool {
					return i == p.Path
				},
			)
			res.MigrationsPaths = utils.AppendIfNotExistInArray(res.MigrationsPaths, p.Schema,
				func(i string) bool {
					return i == p.Schema
				},
			)
		}
	}

	// process sqlc version 2
	if s.Version == 2 {
		for _, p := range s.SQL {
			modelFileName := p.Gen.Go.OutputModelsFileName
			if modelFileName == "" {
				modelFileName = "models.go"
			}

			res.ModelsPaths = utils.AppendIfNotExistInArray(
				res.ModelsPaths,
				filepath.Join(p.Gen.Go.Out, modelFileName),
				func(i string) bool {
					return i == filepath.Join(p.Gen.Go.Out, modelFileName)
				},
			)
			res.QueriesPaths = utils.AppendIfNotExistInArray(res.QueriesPaths, p.Queries,
				func(i string) bool {
					return i == p.Queries
				},
			)
			res.OutPaths = utils.AppendIfNotExistInArray(res.OutPaths, p.Gen.Go.Out,
				func(i string) bool {
					return i == p.Gen.Go.Out
				},
			)
			res.MigrationsPaths = utils.AppendIfNotExistInArray(res.MigrationsPaths, p.Schema,
				func(i string) bool {
					return i == p.Schema
				},
			)
		}
	}

	return res
}

func (p *Package) GetModelPath() string {
	modelFileName := p.OutputModelsFileName
	if modelFileName == "" {
		modelFileName = "models.go"
	}
	return filepath.Join(p.Path, modelFileName)
}

func (p *SqlcSQL) GetModelPath() string {
	modelFileName := p.Gen.Go.OutputModelsFileName
	if modelFileName == "" {
		modelFileName = "models.go"
	}
	return filepath.Join(p.Gen.Go.Out, modelFileName)
}
