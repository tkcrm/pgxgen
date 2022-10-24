package gomodels

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/generator"
	"github.com/tkcrm/pgxgen/internal/structs"
	"github.com/tkcrm/pgxgen/internal/templates"
	"github.com/tkcrm/pgxgen/utils"
	"golang.org/x/tools/imports"
)

type gomodels struct {
	config config.Config
}

func New(cfg config.Config) generator.IGenerator {
	return &gomodels{
		config: cfg,
	}
}

type tmplGoModelsCtx struct {
	Version string
	Package string
	Structs structs.Structs
	Imports string
}

func (s *gomodels) Generate(args []string) error {
	if err := s.generateModels(args); err != nil {
		return err
	}

	fmt.Println("models successfully generated")

	return nil
}

func (s *gomodels) generateModels(args []string) error {
	if len(s.config.Sqlc.Packages) < len(s.config.Pgxgen.GenModels) {
		return fmt.Errorf("sqlc packages should be more or equal pgxgen gen_models")
	}

	for index, p := range s.config.Sqlc.Packages {
		// get models.go from sqlc
		file, err := os.ReadFile(p.GetModelPath())
		if err != nil {
			return err
		}

		_structs := structs.GetStructs(string(file))

		config := s.config.Pgxgen.GenModels[index]
		if err := s.processStructs(config, &_structs); err != nil {
			return err
		}

		goModels, err := s.compileGoModels(config, _structs, p.GetModelPath())
		if err != nil {
			return err
		}
		if err := templates.CompileTemplate(goModels); err != nil {
			return err
		}

		// get all types from ModelsOutputDir
		scalarTypes := make(structs.Types)
		dirItems, err := os.ReadDir(config.GetModelsOutputDir())
		if err != nil {
			return err
		}
		for _, item := range dirItems {
			if item.IsDir() {
				continue
			}
			path := filepath.Join(config.GetModelsOutputDir(), item.Name())

			file, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			for key, value := range structs.GetStructs(string(file)) {
				_structs[key] = value
			}
			for key, value := range s.getScalarTypes(string(file)) {
				scalarTypes[key] = value
			}
		}

		mobxKeystoneModels, err := compileMobxKeystoneModels(s.config.Pgxgen.Version, config, _structs, scalarTypes)
		if err != nil {
			return err
		}
		if err := templates.CompileTemplate(mobxKeystoneModels); err != nil {
			return err
		}

		if config.DeleteSqlcData {
			if err := os.RemoveAll(p.Path); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *gomodels) getScalarTypes(file_models_str string) structs.Types {
	types := make(structs.Types)
	r := bufio.NewReader(strings.NewReader(file_models_str))

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}

		if strings.Contains(line, "struct {") {
			continue
		}
		r := regexp.MustCompile(`^type (\w+) ([^\s]+)`)
		match := r.FindStringSubmatch(line)

		if len(match) == 3 {
			types[match[1]] = structs.TypesParameters{
				Name: match[1],
				Type: match[2],
			}
		}
	}

	return types
}

func (s *gomodels) processStructs(c config.GenModels, st *structs.Structs) error {
	if c.UseUintForIds {
		for _, s := range *st {
			for _, f := range s.Fields {
				if !strings.HasSuffix(f.Name, "ID") {
					continue
				}

				if c.ExistPrefereExceptionsField(s.Name, f.Name) {
					continue
				}

				pointer := strings.HasPrefix(f.Type, "*")
				ftype := strings.ReplaceAll(f.Type, "*", "")
				if !utils.ExistInArray([]string{"int16", "int32", "int64"}, ftype) {
					continue
				}

				switch ftype {
				case "int16":
					f.Type = "uint16"
				case "int32":
					f.Type = "uint32"
				case "int64":
					f.Type = "uint64"
				}

				if pointer {
					f.Type = "*" + f.Type
				}
			}
		}
	}

	// process add fields
	for _, f := range c.AddFields {
		s, ok := (*st)[f.StructName]
		if !ok {
			return fmt.Errorf("struct %s not found in models. use case sensitive names", f.StructName)
		}

		newField := structs.StructField{
			Name: f.FieldName,
			Type: f.Type,
			Tags: make(map[string]string),
		}

		for _, tag := range f.Tags {
			newField.Tags[tag.Name] = tag.Value
		}

		if f.Position == "" {
			f.Position = "start"
		}

		switch f.Position {
		case "start":
			s.Fields = append([]*structs.StructField{&newField}, s.Fields...)
		case "end":
			s.Fields = append(s.Fields, &newField)
		default:
			r := regexp.MustCompile(`after ([\w\.]+)`)
			if !r.MatchString(f.Position) {
				return fmt.Errorf("unavailable position %s for struct %s", f.Position, f.StructName)
			}
			match := r.FindStringSubmatch(f.Position)
			position_after := match[1]
			existFieldIndex := s.ExistFieldIndex(position_after)
			if position_after != "" && existFieldIndex == -1 {
				return fmt.Errorf("field %s does not exist in struct %s", position_after, f.StructName)
			}
			s.Fields = append(s.Fields[:existFieldIndex+1], append([]*structs.StructField{&newField}, s.Fields[existFieldIndex+1:]...)...)
		}
	}

	// process update all struct fields by field name
	for _, param := range c.UpdateAllStructFields.ByField {
		for _, s := range *st {
			for _, field := range s.Fields {
				if field.Name != param.FieldName {
					continue
				}

				if param.NewFieldName != "" {
					field.Name = param.NewFieldName
				}

				if param.NewType != "" {
					field.Type = param.NewType
				}

				if !param.MatchWithCurrentTags {
					field.Tags = map[string]string{}
				}

				for _, tag := range param.Tags {
					field.Tags[tag.Name] = tag.Value
				}
			}
		}
	}

	// process update all struct fields by type
	for _, param := range c.UpdateAllStructFields.ByType {
		for _, s := range *st {
			for _, field := range s.Fields {
				if field.Type != param.Type {
					continue
				}
				field.Type = param.NewType

				if !param.MatchWithCurrentTags {
					field.Tags = map[string]string{}
				}

				for _, tag := range param.Tags {
					field.Tags[tag.Name] = tag.Value
				}
			}
		}
	}

	// process update fields
	for _, f := range c.UpdateFields {
		s, ok := (*st)[f.StructName]
		if !ok {
			return fmt.Errorf("struct %s not found in models. use case sensitive names", f.StructName)
		}

		existFieldIndex := s.ExistFieldIndex(f.FieldName)
		if existFieldIndex == -1 {
			return fmt.Errorf("field %s does not exist in struct %s", f.FieldName, f.StructName)
		}

		existField := s.Fields[existFieldIndex]

		params := f.NewParameters
		if params.Name != "" {
			existField.Name = params.Name
		}

		if params.Type != "" {
			existField.Type = params.Type
		}

		if !params.MatchWithCurrentTags {
			existField.Tags = map[string]string{}
		}

		for _, tag := range params.Tags {
			existField.Tags[tag.Name] = tag.Value
		}
	}

	// process delete fields
	for _, item := range c.DeleteFields {
		s, ok := (*st)[item.StructName]
		if !ok {
			return fmt.Errorf("struct %s not found in models. use case sensitive names", item.StructName)
		}

		for _, name := range item.FieldNames {
			existFieldIndex := s.ExistFieldIndex(name)
			if existFieldIndex == -1 {
				return fmt.Errorf("field %s does not exist in struct %s", name, item.StructName)
			} else {
				s.Fields = append(s.Fields[:existFieldIndex], s.Fields[existFieldIndex+1:]...)
			}
		}
	}

	return nil
}

func (s *gomodels) compileGoModels(c config.GenModels, st structs.Structs, path string) (*templates.CompileData, error) {
	if c.ModelsOutputDir == "" {
		return nil, fmt.Errorf("config error: undefined models_output_dir")
	}

	cdata := templates.CompileData{
		OutputDir:      c.GetModelsOutputDir(),
		OutputFileName: c.GetModelsOutputFileName(),
	}

	pnsplit := strings.Split(c.ModelsOutputDir, "/")
	packageName := pnsplit[len(pnsplit)-1]

	if c.ModelsPackageName != "" {
		packageName = c.ModelsPackageName
	}

	tmpl := template.Must(
		template.New("goModels").
			ParseFS(
				templates.Templates,
				"templates/models.go.tmpl",
			),
	)

	allImports := []string{}
	for _, s := range st {
		for _, i := range s.Imports {
			if !utils.ExistInArray(allImports, i) {
				allImports = append(allImports, i)
			}
		}
	}

	tctx := tmplGoModelsCtx{
		Version: s.config.Pgxgen.Version,
		Package: packageName,
		Structs: st,
		Imports: strings.Join(allImports, "\n"),
	}

	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	err := tmpl.ExecuteTemplate(w, "modelsFile", &tctx)
	w.Flush()
	if err != nil {
		return nil, fmt.Errorf("execte template error: %s", err.Error())
	}

	cdata.Data, err = imports.Process("", b.Bytes(), nil)
	if err != nil {
		fmt.Println(b.String())
		return nil, fmt.Errorf("formate data error: %s", err.Error())
	}

	return &cdata, nil
}
