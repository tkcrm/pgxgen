package config

import "path/filepath"

type Sqlc struct {
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

func (p *Package) GetModelPath() string {
	modelFileName := p.OutputModelsFileName
	if modelFileName == "" {
		modelFileName = "models.go"
	}
	return filepath.Join(p.Path, modelFileName)
}
