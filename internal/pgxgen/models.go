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

type tmplGoModelsCtx struct {
	Package string
	Structs Structs
	Imports string
}

type structParameters struct {
	Name    string
	Imports []string
	Fields  []*structField
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
	tags map[string]string
}

func (s *structField) GetGoTag() string {
	tag := ""

	keys := make([]string, 0, len(s.tags))
	for k := range s.tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if tag != "" {
			tag += " "
		}
		tag += fmt.Sprintf("%s:\"%s\"", k, s.tags[k])
	}

	return tag
}

type typesParameters struct {
	Name string
	Type string
}

type Types map[string]typesParameters
type Structs map[string]*structParameters
type StructSlice []*structParameters

func (st *StructSlice) ExistStructIndex(name string) (int, *structParameters) {
	for index, s := range *st {
		if s.Name == name {
			return index, s
		}
	}
	return -1, nil
}

func (st *StructSlice) Sort(priorityNames ...string) error {

	names := make([]string, 0, len(*st))
	for _, name := range priorityNames {
		existStructIndex, _ := st.ExistStructIndex(name)
		if existStructIndex == -1 {
			return fmt.Errorf("sort error: undefined struct %s", name)
		}

		names = append(names, name)
	}

	notPriorityNames := make([]string, 0, len(*st)-len(priorityNames))
	for _, v := range *st {
		if utils.ExistInStringArray(names, v.Name) {
			continue
		}
		notPriorityNames = append(notPriorityNames, v.Name)
	}
	sort.Strings(notPriorityNames)

	names = append(names, notPriorityNames...)

	sorted := make(StructSlice, 0, len(names))
	for _, n := range names {
		for _, s := range *st {
			if s.Name == n {
				sorted = append(sorted, s)
			}
		}
	}

	*st = sorted

	return nil
}

func ConvertStructsToSlice(st Structs) StructSlice {
	res := make(StructSlice, 0, len(st))

	for _, s := range st {
		res = append(res, s)
	}

	return res
}

type compileData struct {
	data           []byte
	outputDir      string
	outputFileName string
	afterHook      func() error
}

func generateModels(args []string, c config.Config) error {

	if len(c.Sqlc.Packages) < len(c.Pgxgen.GenModels) {
		return fmt.Errorf("sqlc packages should be more or equal pgxgen gen_models")
	}

	for index, p := range c.Sqlc.Packages {

		// get models.go from sqlc
		file, err := os.ReadFile(p.GetModelPath())
		if err != nil {
			return err
		}

		structs := getStructs(string(file))

		config := c.Pgxgen.GenModels[index]
		if err := processStructs(config, &structs); err != nil {
			return err
		}

		goModels, err := compileGoModels(config, structs, p.GetModelPath())
		if err != nil {
			return err
		}
		if err := compileTemplate(goModels); err != nil {
			return err
		}

		// get all types from ModelsOutputDir
		scalarTypes := make(Types)
		dirItems, err := os.ReadDir(config.GetModelsOutputDir())
		for _, item := range dirItems {
			if item.IsDir() {
				continue
			}
			path := filepath.Join(config.GetModelsOutputDir(), item.Name())

			file, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			for key, value := range getStructs(string(file)) {
				structs[key] = value
			}
			for key, value := range getScalarTypes(string(file)) {
				scalarTypes[key] = value
			}
		}

		mobxKeystoneModels, err := compileMobxKeystoneModels(config, structs, scalarTypes)
		if err != nil {
			return err
		}
		if err := compileTemplate(mobxKeystoneModels); err != nil {
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

func getImports(file_models_str string) []string {
	res := ""
	r := regexp.MustCompile(`(?s)import \(?([^)]+)\)?`)
	match := r.FindAllStringSubmatch(file_models_str, -1)
	if len(match) == 1 {
		res = match[0][1]
	}
	return strings.Split(strings.ReplaceAll(res, "\r\n", "\n"), "\n")
}

func getScalarTypes(file_models_str string) Types {
	types := make(Types)
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
			types[match[1]] = typesParameters{
				Name: match[1],
				Type: match[2],
			}
		}
	}

	return types
}

func getStructs(file_models_str string) Structs {
	r := bufio.NewReader(strings.NewReader(file_models_str))

	structs := make(Structs)

	currentStruct := &structParameters{
		Imports: getImports(file_models_str),
	}
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
			currentStruct = &structParameters{}
			continue
		}

		reStartStruct := regexp.MustCompile(`type (\w+) struct {`)
		matches := reStartStruct.FindAllStringSubmatch(line, -1)
		if len(matches) == 1 {
			currentStruct = &structParameters{Name: matches[0][1], Fields: make([]*structField, 0)}
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
	}

	// process update all struct fields by field
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
					field.tags = map[string]string{}
				}

				for _, tag := range param.Tags {
					field.tags[tag.Name] = tag.Value
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
					field.tags = map[string]string{}
				}

				for _, tag := range param.Tags {
					field.tags[tag.Name] = tag.Value
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
	}

	return nil
}

func compileGoModels(c config.GenModels, st Structs, path string) (*compileData, error) {
	if c.ModelsOutputDir == "" {
		return nil, fmt.Errorf("config error: undefined models_output_dir")
	}

	cdata := compileData{
		outputDir:      c.GetModelsOutputDir(),
		outputFileName: c.GetModelsOutputFileName(),
	}

	pnsplit := strings.Split(c.ModelsOutputDir, "/")
	packageName := pnsplit[len(pnsplit)-1]

	if c.ModelsPackageName != "" {
		packageName = c.ModelsPackageName
	}

	tmpl := template.Must(
		template.New("goModels").
			ParseFS(
				templates,
				"templates/models.go.tmpl",
			),
	)

	allImports := []string{}
	for _, s := range st {
		for _, i := range s.Imports {
			if !utils.ExistInStringArray(allImports, i) {
				allImports = append(allImports, i)
			}
		}
	}

	tctx := tmplGoModelsCtx{
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

	cdata.data, err = imports.Process("", b.Bytes(), nil)
	if err != nil {
		fmt.Println(b.String())
		return nil, fmt.Errorf("formate data error: %s", err.Error())
	}

	return &cdata, nil
}

func compileTemplate(d *compileData) error {

	if d.data == nil && len(d.data) == 0 {
		return fmt.Errorf("compile template error: data is undefined")
	}

	if d.outputDir == "" {
		return fmt.Errorf("compile template error: output dir is empty")
	}

	if d.outputFileName == "" {
		return fmt.Errorf("compile template error: output file name is empty")
	}

	if err := os.MkdirAll(d.outputDir, os.ModePerm); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(d.outputDir, d.outputFileName), d.data, os.ModePerm); err != nil {
		return fmt.Errorf("write error: %s", err.Error())
	}

	fmt.Println("successfully generated in:", filepath.Join(d.outputDir, d.outputFileName))

	if d.afterHook != nil {
		if err := d.afterHook(); err != nil {
			return err
		}
	}

	return nil
}
