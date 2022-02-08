package pgxgen

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/utils"
	"golang.org/x/tools/imports"
)

type structParameters struct {
	Name   string
	Fields []*structField
}

func (s *structParameters) existFieldIndex(name string) int {
	for index, f := range s.Fields {
		if f.Name == name {
			return index
		}
	}
	return -1
}

type structField struct {
	Name string
	Type string
	Tag  string
	tags map[string]string
}

func (s *structField) convertTags() {
	s.Tag = ""

	keys := make([]string, 0, len(s.tags))
	for k := range s.tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if s.Tag != "" {
			s.Tag += " "
		}
		s.Tag += fmt.Sprintf("%s:\"%s\"", k, s.tags[k])
	}
}

type Structs map[string]structParameters

func generateModels(args []string, c config.Config) error {

	if len(c.Sqlc.Packages) < len(c.Pgxgen.GenModels) {
		return fmt.Errorf("sqlc packages should be more or equal pgxgen gen_models")
	}

	for index, p := range c.Sqlc.Packages {
		file, err := os.ReadFile(p.GetModelPath())
		if err != nil {
			return err
		}

		structs := getStructs(string(file))

		config := c.Pgxgen.GenModels[index]
		if err := processStructs(config, &structs); err != nil {
			return err
		}

		imports := getImports(string(file))

		if err := compileTemplate(config, structs, imports, p.GetModelPath()); err != nil {
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

func getImports(file_models_str string) string {
	res := ""
	r := regexp.MustCompile(`(?s)import \(?([^)]+)\)?`)
	match := r.FindAllStringSubmatch(file_models_str, -1)
	if len(match) == 1 {
		res = match[0][1]
	}
	return res
}

func getStructs(file_models_str string) Structs {
	r := bufio.NewReader(strings.NewReader(file_models_str))

	structs := make(Structs)

	currentStruct := structParameters{}
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}

		reCloseStruct := regexp.MustCompile(`^\}`)
		if reCloseStruct.MatchString(line) {
			structs[currentStruct.Name] = currentStruct
			currentStruct = structParameters{}
			continue
		}

		reStartStruct := regexp.MustCompile(`type (\w+) struct {`)
		matches := reStartStruct.FindAllStringSubmatch(line, -1)
		if len(matches) == 1 {
			currentStruct = structParameters{Name: matches[0][1], Fields: make([]*structField, 0)}
			continue
		}

		// parse fields
		if currentStruct.Name != "" {
			reField := regexp.MustCompile(`\s*(?P<name>[\w\.]+)\s+(?P<type>[\w\*\.]+)\s+(?P<tags>\x60.+\x60)?`)
			match := reField.FindStringSubmatch(line)
			if len(match) == 0 {
				continue
			}
			field := structField{tags: make(map[string]string)}
			for index, name := range reField.SubexpNames() {
				if index != 0 && name != "" {
					switch name {
					case "name":
						field.Name = match[index]
					case "type":
						field.Type = match[index]
					case "tags":
						reTags := regexp.MustCompile(`(\w+):\"(\w+)\"`)
						match := reTags.FindAllStringSubmatch(match[index], -1)
						for _, m := range match {
							field.tags[m[1]] = m[2]
						}
					}
				}
			}
			if field.Name != "" {
				currentStruct.Fields = append(currentStruct.Fields, &field)
			}
		}
	}

	return structs
}

func processStructs(c config.GenModels, st *Structs) error {

	if c.PrefereUintForIds {
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
				if !utils.ExistInStringArray([]string{"int16", "int32", "int64"}, ftype) {
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
			(*st)[s.Name] = s
		}
	}

	// process add fields
	for _, f := range c.AddFields {
		s, ok := (*st)[f.StructName]
		if !ok {
			return fmt.Errorf("struct %s not found in models. use case sensitive names", f.StructName)
		}

		newField := structField{
			Name: f.FieldName,
			Type: f.Type,
			tags: make(map[string]string),
		}

		for _, tag := range f.Tags {
			newField.tags[tag.Name] = tag.Value
		}

		if f.Position == "" {
			f.Position = "start"
		}

		switch f.Position {
		case "start":
			s.Fields = append([]*structField{&newField}, s.Fields...)
		case "end":
			s.Fields = append(s.Fields, &newField)
		default:
			r := regexp.MustCompile(`after ([\w\.]+)`)
			if !r.MatchString(f.Position) {
				return fmt.Errorf("unavailable position %s for struct %s", f.Position, f.StructName)
			}
			match := r.FindStringSubmatch(f.Position)
			position_after := match[1]
			existFieldIndex := s.existFieldIndex(position_after)
			if position_after != "" && existFieldIndex == -1 {
				return fmt.Errorf("field %s does not exist in struct %s", position_after, f.StructName)
			}
			s.Fields = append(s.Fields[:existFieldIndex+1], append([]*structField{&newField}, s.Fields[existFieldIndex+1:]...)...)
		}

		(*st)[f.StructName] = s
	}

	// process update fields
	for _, f := range c.UpdateFields {
		s, ok := (*st)[f.StructName]
		if !ok {
			return fmt.Errorf("struct %s not found in models. use case sensitive names", f.StructName)
		}

		existFieldIndex := s.existFieldIndex(f.FieldName)
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
			existField.tags = map[string]string{}
		}

		for _, tag := range params.Tags {
			existField.tags[tag.Name] = tag.Value
		}

		(*st)[f.StructName] = s
	}

	// process delete fields
	for _, item := range c.DeleteFields {
		s, ok := (*st)[item.StructName]
		if !ok {
			return fmt.Errorf("struct %s not found in models. use case sensitive names", item.StructName)
		}

		for _, name := range item.FieldNames {
			existFieldIndex := s.existFieldIndex(name)
			if existFieldIndex == -1 {
				return fmt.Errorf("field %s does not exist in struct %s", name, item.StructName)
			} else {
				s.Fields = append(s.Fields[:existFieldIndex], s.Fields[existFieldIndex+1:]...)
			}
		}

		(*st)[item.StructName] = s
	}

	return nil
}

func compileTemplate(c config.GenModels, st Structs, importstr, path string) error {

	if c.ModelsOutputDir == "" {
		return fmt.Errorf("config error: undefined models_output_dir")
	}

	if c.ModelsOutputFilename != "" && !strings.HasSuffix(c.ModelsOutputFilename, ".go") {
		c.ModelsOutputFilename += ".go"
	}

	if string(c.ModelsOutputDir[len(c.ModelsOutputDir)-1]) == "/" {
		c.ModelsOutputDir = c.ModelsOutputDir[:len(c.ModelsOutputDir)]
	}

	pnsplit := strings.Split(c.ModelsOutputDir, "/")
	packageName := pnsplit[len(pnsplit)-1]

	if c.ModelsPackageName != "" {
		packageName = c.ModelsPackageName
	}

	tmpl := template.Must(
		template.New("table").
			ParseFS(
				templates,
				"templates/*.tmpl",
			),
	)

	for _, s := range st {
		for _, f := range s.Fields {
			f.convertTags()
		}
	}

	tctx := tmplCtx{
		Package: packageName,
		Structs: st,
		Imports: importstr,
	}

	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	err := tmpl.ExecuteTemplate(w, "modelsFile", &tctx)
	w.Flush()
	if err != nil {
		return fmt.Errorf("execte template error: %s", err.Error())
	}

	formated, err := imports.Process("", b.Bytes(), nil)
	if err != nil {
		fmt.Println(b.String())
		return fmt.Errorf("formate data error: %s", err.Error())
	}

	if err := os.MkdirAll(c.ModelsOutputDir, os.ModePerm); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(c.ModelsOutputDir, c.ModelsOutputFilename), formated, 0644); err != nil {
		return fmt.Errorf("write error: %s", err.Error())
	}

	return nil
}
