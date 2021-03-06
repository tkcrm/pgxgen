package gomodels

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/structs"
	"github.com/tkcrm/pgxgen/internal/templates"
)

type tmplKeystoneCtx struct {
	Structs                  structs.StructSlice
	Imports                  map[string][]string
	ImportTypes              map[string][]string
	DecoratorModelNamePrefix string
	ExportModelSuffix        string
	WithSetter               bool
}

func compileMobxKeystoneModels(c config.GenModels, st structs.Structs, sct structs.Types) (*templates.CompileData, error) {
	config := c.ExternalModels.Keystone

	if config.OutputDir == "" {
		return nil, fmt.Errorf("compile mobx keystone error: undefined output dir")
	}

	if config.OutputFileName == "" {
		config.OutputFileName = "models.ts"
	}

	cdata := templates.CompileData{
		OutputDir:      config.OutputDir,
		OutputFileName: config.OutputFileName,
	}

	funcs := templates.DefaultTmplFuncs
	templates.TmplAddFunc(funcs, "getType", func(t string) string {

		typeWrap, tp := getKeystoneType(c, st, sct, t)

		if config.WithSetter {
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
	if err := structs.Sort(strings.Split(config.Sort, ",")...); err != nil {
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
		DecoratorModelNamePrefix: config.DecoratorModelNamePrefix,
		ExportModelSuffix:        config.ExportModelSuffix,
		WithSetter:               true,
	}

	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	err := tmpl.ExecuteTemplate(w, "keystoneModelsFile", &tctx)
	w.Flush()
	if err != nil {
		return nil, fmt.Errorf("execte template error: %s", err.Error())
	}

	cdata.Data = b.Bytes()

	if config.PrettierCode {
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

func getKeystoneType(c config.GenModels, st structs.Structs, sct structs.Types, t string) (typeWrap string, tp string) {

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
			tp = fmt.Sprintf("types.model(%s%s)", t, c.ExternalModels.Keystone.ExportModelSuffix)
		} else if oksct {
			typeWrap, tp = getKeystoneType(c, st, sct, existScalarItem.Type)
		} else {
			tp = "types.unchecked()"
			fmt.Println("undefined type:", t)
		}
	}

	return typeWrap, tp
}
