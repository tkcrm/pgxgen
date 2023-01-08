package keystone

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/generator"
	"github.com/tkcrm/pgxgen/internal/structs"
	"github.com/tkcrm/pgxgen/internal/templates"
)

type keystone struct {
	config config.Config
}

func New(cfg config.Config) generator.IGenerator {
	return &keystone{
		config: cfg,
	}
}

func (s *keystone) Generate(args []string) error {
	if err := s.generateKeystone(args); err != nil {
		return err
	}

	fmt.Println("models successfully generated")

	return nil
}

func (s *keystone) generateKeystone(args []string) error {
	if s.config.Sqlc.Version > 2 || s.config.Sqlc.Version < 1 {
		return fmt.Errorf("unsupported sqlc version: %d", s.config.Sqlc.Version)
	}

	if len(s.config.Sqlc.Packages) < len(s.config.Pgxgen.GenModels) {
		return fmt.Errorf("sqlc packages should be more or equal pgxgen gen_models")
	}

	for index, modelsFilePath := range s.config.Sqlc.GetModelPaths() {
		// get models.go path
		if s.config.Pgxgen.SqlcModels.OutputDir != "" {
			fileName := "models.go"
			if s.config.Pgxgen.SqlcModels.OutputFilename != "" {
				fileName = s.config.Pgxgen.SqlcModels.OutputFilename
			}

			modelsFilePath = filepath.Join(s.config.Pgxgen.SqlcModels.OutputDir, fileName)
		}

		// get models.go file content
		fileContent, err := os.ReadFile(modelsFilePath)
		if err != nil {
			return err
		}

		// get structs from go file
		modelStructs := structs.GetStructs(string(fileContent))

		if len(s.config.Pgxgen.GenKeystoneFromStruct) < index+1 {
			return fmt.Errorf("undefined gen keystone config")
		}

		config := s.config.Pgxgen.GenKeystoneFromStruct[index]

		// get all types from ModelsOutputDir
		scalarTypes := make(structs.Types)
		dirItems, err := os.ReadDir(filepath.Dir(modelsFilePath))
		if err != nil {
			return err
		}

		allStructs := make(structs.Structs)

		for _, item := range dirItems {
			if item.IsDir() {
				continue
			}
			path := filepath.Join(filepath.Dir(modelsFilePath), item.Name())

			file, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			for key, value := range structs.GetStructs(string(file)) {
				allStructs[key] = value
			}

			for key, value := range s.getScalarTypes(string(file)) {
				scalarTypes[key] = value
			}
		}

		structs.FillMissedTypes(allStructs, modelStructs, scalarTypes)

		for _, modelName := range config.SkipModels {
			delete(modelStructs, modelName)
		}

		mobxKeystoneModels, err := compileMobxKeystoneModels(s.config.Pgxgen.Version, config, modelStructs, scalarTypes)
		if err != nil {
			return err
		}
		if err := templates.CompileTemplate(mobxKeystoneModels); err != nil {
			return err
		}
	}

	return nil
}

func (s *keystone) getScalarTypes(file_models_str string) structs.Types {
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

func compileMobxKeystoneModels(ver string, cfg config.GenKeystoneFromStruct, st structs.Structs, sct structs.Types) (*templates.CompileData, error) {
	if cfg.OutputDir == "" {
		return nil, fmt.Errorf("compile mobx keystone error: undefined output dir")
	}

	if cfg.OutputFileName == "" {
		cfg.OutputFileName = "models.ts"
	}

	cdata := templates.CompileData{
		OutputDir:      cfg.OutputDir,
		OutputFileName: cfg.OutputFileName,
	}

	funcs := templates.DefaultTmplFuncs
	templates.TmplAddFunc(funcs, "getType", func(t string) string {
		typeWrap, tp := getKeystoneType(cfg, st, sct, t)

		if cfg.WithSetter {
			typeWrap += ".withSetter()"
		}

		return fmt.Sprintf(typeWrap, tp)
	})

	templates.TmplAddFunc(funcs, "exist_field_id", func(structName, fieldName string) bool {
		for _, s := range st {
			if s.Name != structName {
				continue
			}
			for _, f := range s.Fields {
				if strings.ToLower(f.Name) == fieldName {
					return true
				}
			}
		}

		return false
	})

	tmpl := template.Must(
		template.New("mobxKeystoneModels").
			Funcs(funcs).
			ParseFS(
				templates.Templates,
				"templates/keystone-model.go.tmpl",
			),
	)

	mobxKeystoneImports := []string{
		"Model", "tProp", "types", "model", "draft", "modelAction",
		"prop", "clone", "Draft",
	}

	// for _, s := range st {
	// 	for _, f := range s.Fields {
	// 		if utils.ExistInArray([]string{"int64", "uint64"}, f.Type) &&
	// 			!utils.ExistInArray(mobxKeystoneImports, "stringToBigIntTransform") {
	// 			mobxKeystoneImports = append(mobxKeystoneImports, "stringToBigIntTransform")
	// 			break
	// 		}
	// 	}
	// }

	structs := structs.ConvertStructsToSlice(st)
	if err := structs.Sort(strings.Split(cfg.Sort, ",")...); err != nil {
		return nil, err
	}

	tctx := tmplKeystoneCtx{
		Structs: structs,
		Imports: map[string][]string{
			"mobx-keystone": mobxKeystoneImports,
			"mobx":          {"computed", "observable"},
		},
		ImportTypes: map[string][]string{
			"@tkcrm/ui": {"FormInstance"},
		},
		DecoratorModelNamePrefix: cfg.DecoratorModelNamePrefix,
		ExportModelSuffix:        cfg.ExportModelSuffix,
		WithSetter:               true,
		Version:                  ver,
	}

	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	err := tmpl.ExecuteTemplate(w, "keystoneModelsFile", &tctx)
	w.Flush()
	if err != nil {
		return nil, fmt.Errorf("execte template error: %s", err.Error())
	}

	cdata.Data = b.Bytes()

	if cfg.PrettierCode {
		cdata.AfterHook = func() error {
			fmt.Println("prettier generated models ...")
			cmd := exec.Command("npx", "prettier", "--write", filepath.Join(cdata.OutputDir, cdata.OutputFileName))
			stdout, err := cmd.Output()
			if err != nil {
				fmt.Println(err.Error())
				return nil
			}
			if len(stdout) > 0 {
				fmt.Println(string(stdout))
			}
			return nil
		}
	}

	return &cdata, nil
}

func getKeystoneType(cfg config.GenKeystoneFromStruct, st structs.Structs, sct structs.Types, t string) (typeWrap string, tp string) {
	typeWrap = "tProp(%s)"
	typeWrapUnchecked := "prop<%s>"

	isNullable := strings.Contains(t, "Null") || strings.HasPrefix(t, "*")
	if isNullable {
		t = strings.ReplaceAll(t, "*", "")
		typeWrap = "tProp(types.maybe(%s))"
		typeWrapUnchecked = "prop<%s | undefined>"
	}

	tp = ""
	switch t {
	case "int", "int8", "int16", "int32",
		"uint", "uint8", "uint16", "uint32":
		tp = "types.integer"
		if !isNullable {
			tp += ",0"
		}
	// case "int64", "uint64":
	// 	tp = "types.integer"
	// 	if !isNullable {
	// 		tp += ",0"
	// 	}
	case "int64", "uint64":
		tp = "bigint"
		if !isNullable {
			typeWrap = typeWrapUnchecked + "(0n)"
		} else {
			typeWrap = typeWrapUnchecked + "()"
		}
	case "float32", "float64":
		tp = "types.number"
		if !isNullable {
			tp += ",0"
		}
	case "string":
		tp = "types.string"
		if !isNullable {
			tp += ",\"\""
		}
	case "bool":
		tp = "types.boolean"
		if !isNullable {
			tp += ",false"
		}
	case "bun.NullTime", "time.Time", "pgtype.Time":
		tp = "types.dateString"
		if !isNullable {
			tp += ",\"\""
		}
	case "uuid.UUID", "uuid.NullUUID":
		tp = "types.string"
		if !isNullable {
			tp += ",\"\""
		}
	case "map[string]interface{}", "pgtype.JSONB":
		tp = "Record<string, any>"
		typeWrap = typeWrapUnchecked + "({})"
	default:
		_, okst := st[t]
		existScalarItem, oksct := sct[t]
		if okst {
			// var defaultValue string
			// if !isNullable {
			// 	defaultValue = fmt.Sprintf(", new %s%s({})", t, c.ExternalModels.Keystone.ExportModelSuffix)
			// }
			// tp = fmt.Sprintf("types.model(%s%s)%s", t, c.ExternalModels.Keystone.ExportModelSuffix, defaultValue)
			tp = fmt.Sprintf("types.model(%s%s)", t, cfg.ExportModelSuffix)
		} else if oksct {
			typeWrap, tp = getKeystoneType(cfg, st, sct, existScalarItem.Type)
		} else {
			tp = "types.unchecked()"
			fmt.Println("undefined type:", t)
		}
	}

	return typeWrap, tp
}
