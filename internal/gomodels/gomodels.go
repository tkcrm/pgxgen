package gomodels

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/tkcrm/modules/pkg/templates"
	"github.com/tkcrm/pgxgen/internal/assets"
	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/generator"
	"github.com/tkcrm/pgxgen/internal/structs"
	"github.com/tkcrm/pgxgen/pkg/logger"
	"github.com/tkcrm/pgxgen/utils"
)

type gomodels struct {
	logger logger.Logger
	config config.Config
}

func New(logger logger.Logger, cfg config.Config) generator.IGenerator {
	return &gomodels{
		logger: logger,
		config: cfg,
	}
}

func (s *gomodels) Generate(_ context.Context, args []string) error {
	if len(s.config.Pgxgen.GenModels) == 0 {
		return nil
	}

	s.logger.Infof("generate go models")
	timeStart := time.Now()

	for _, genStructsCfg := range s.config.Pgxgen.GenModels {
		if err := s.generateModels(genStructsCfg); err != nil {
			return err
		}
	}

	s.logger.Infof("go models successfully generated in: %s", time.Since(timeStart))

	return nil
}

func (s *gomodels) generateModels(cfg config.GenModels) error {
	// validate input data
	if cfg.InputFilePath == "" &&
		cfg.InputDir == "" {
		return fmt.Errorf("empty input file path and input dir params")
	}

	if cfg.InputFilePath != "" &&
		cfg.InputDir != "" {
		return fmt.Errorf("you can use input_file_path or input_dir at the same time")
	}

	// validate output data
	if cfg.OutputDir == "" {
		return fmt.Errorf("empty output_dir param")
	}

	// validate output data
	if cfg.OutputFileName == "" {
		return fmt.Errorf("empty output_file_name param")
	}

	filePaths := []string{}
	if cfg.InputFilePath != "" {
		path, err := filepath.Abs(cfg.InputFilePath)
		if err != nil {
			return err
		}

		filePaths = append(filePaths, path)
	}

	// get file names for dir
	if cfg.InputDir != "" {
		dirItems, err := os.ReadDir(cfg.InputDir)
		if err != nil {
			return err
		}

		for _, item := range dirItems {
			if item.IsDir() {
				continue
			}

			path, err := filepath.Abs(filepath.Join(cfg.InputDir, item.Name()))
			if err != nil {
				return err
			}

			if slices.Contains(filePaths, path) {
				continue
			}

			filePaths = append(filePaths, path)
		}
	}

	if len(filePaths) == 0 {
		return fmt.Errorf("empty file paths list")
	}

	for index, filePath := range filePaths {
		// get structs from go file
		_structs := structs.GetStructsByFilePath(filePath)

		// filter structs by exclude_structs params
		if len(cfg.ExcludeStructs) > 0 {
			for _, param := range cfg.ExcludeStructs {
				for key, value := range _structs {
					if value.Name == param.StructName {
						delete(_structs, key)
					}
				}
			}
		}

		// filter structs by exclude_structs params
		if len(cfg.IncludeStructs) > 0 {
			for key, value := range _structs {
				for _, param := range cfg.IncludeStructs {
					if value.Name != param.StructName {
						delete(_structs, key)
					}
				}
			}
		}

		if len(_structs) == 0 {
			return fmt.Errorf("empty filtered structs lists")
		}

		config := s.config.Pgxgen.GenModels[index]
		if err := s.processStructs(config, &_structs); err != nil {
			return fmt.Errorf("processStructs error: %w", err)
		}

		if err := s.compileGoModels(config, _structs, filePath); err != nil {
			return fmt.Errorf("compileGoModels error: %w", err)
		}

		// get all types from ModelsOutputDir
		// scalarTypes := make(structs.Types)
		dirItems, err := os.ReadDir(config.GetModelsOutputDir())
		if err != nil {
			return fmt.Errorf("read dir error: %w", err)
		}

		for _, item := range dirItems {
			if item.IsDir() {
				continue
			}
			path := filepath.Join(config.GetModelsOutputDir(), item.Name())

			// file, err := utils.ReadFile(path)
			// if err != nil {
			// 	return fmt.Errorf("read file error: %w", err)
			// }

			for key, value := range structs.GetStructsByFilePath(path) {
				_structs[key] = value
			}
			// for key, value := range s.getScalarTypes(string(file)) {
			// 	scalarTypes[key] = value
			// }
		}

		if config.DeleteOriginalFiles {
			if err := os.RemoveAll(filePath); err != nil {
				return fmt.Errorf("delete original files error: %w", err)
			}
		}
	}

	return nil
}

// func (s *gomodels) getScalarTypes(fileContent string) structs.Types {
// 	types := make(structs.Types)
// 	r := bufio.NewReader(strings.NewReader(fileContent))

// 	for {
// 		line, err := r.ReadString('\n')
// 		if err != nil {
// 			if err == io.EOF {
// 				break
// 			}
// 			s.logger.Fatal(err)
// 		}

// 		if strings.Contains(line, "struct {") {
// 			continue
// 		}

// 		r := regexp.MustCompile(`^type (\w+) ([^\s]+)`)
// 		match := r.FindStringSubmatch(line)

// 		if len(match) == 3 {
// 			types[match[1]] = structs.TypesParameters{
// 				Name: match[1],
// 				Type: match[2],
// 			}
// 		}
// 	}

// 	return types
// }

func (s *gomodels) processStructs(c config.GenModels, st *structs.Structs) error {
	// rename structs
	if len(c.Rename) > 0 {
		for key, s := range *st {
			newName, existNewName := c.Rename[s.Name]
			if !existNewName {
				continue
			}

			(*st)[key].Name = newName
		}
	}

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
				if !slices.Contains([]string{"int16", "int32", "int64"}, ftype) {
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

func (s *gomodels) compileGoModels(c config.GenModels, st structs.Structs, path string) error {
	if c.OutputDir == "" {
		return fmt.Errorf("config error: undefined output_dir")
	}

	pnsplit := strings.Split(c.OutputDir, "/")
	packageName := pnsplit[len(pnsplit)-1]

	if c.PackageName != "" {
		packageName = c.PackageName
	}

	allImports := []string{}
	for _, item := range c.Imports {
		splitted := strings.Split(item, " ")
		if len(splitted) == 2 {
			item = fmt.Sprintf("%s \"%s\"", splitted[0], splitted[1])
		}

		if len(splitted) == 1 {
			item = fmt.Sprintf("\"%s\"", splitted[0])
		}

		if !slices.Contains(allImports, item) {
			allImports = append(allImports, item)
		}
	}

	for _, s := range st {
		for _, i := range s.Imports {
			if !slices.Contains(allImports, i) {
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

	tpl := templates.New()
	compiledRes, err := tpl.Compile(templates.CompileParams{
		TemplateName: "modelsFile",
		TemplateType: templates.TextTemplateType,
		FS:           assets.TemplatesFS,
		FSPaths: []string{
			"templates/models.go.tmpl",
		},
		Data: tctx,
	})
	if err != nil {
		return fmt.Errorf("tpl.Compile error: %w", err)
	}

	compiledRes, err = utils.UpdateGoImports(compiledRes)
	if err != nil {
		return fmt.Errorf("UpdateGoImports error: %w", err)
	}

	if err := utils.SaveFile(c.OutputDir, c.OutputFileName, compiledRes); err != nil {
		return fmt.Errorf("save file error: %w", err)
	}

	return nil
}
