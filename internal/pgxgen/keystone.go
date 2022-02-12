package pgxgen

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/gobeam/stringy"
	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/utils"
)

type tmplKeystoneCtx struct {
	Structs                  []structParameters
	Imports                  map[string][]string
	DecoratorModelNamePrefix string
	ExportModelSuffix        string
	WithSetter               bool
}

func compileMobxKeystoneModels(c config.GenModels, st Structs, sct Types) (*compileData, error) {
	config := c.ExternalModels.Keystone

	if config.OutputDir == "" {
		return nil, fmt.Errorf("compile mobx keystone error: undefined output dir")
	}

	if config.OutputFileName == "" {
		config.OutputFileName = "models.ts"
	}

	cdata := compileData{
		outputDir:      config.OutputDir,
		outputFileName: config.OutputFileName,
	}

	funcs := template.FuncMap{
		"lower": strings.ToLower,
		"snake_case": func(str string) string {
			return stringy.New(str).SnakeCase().ToLower()
		},
		"lcfirst": func(str string) string {
			if str == "ID" {
				return "id"
			}
			return stringy.New(str).LcFirst()
		},
		"join": strings.Join,
		"getType": func(t string) string {

			typeWrap, tp := getKeystoneType(c, st, sct, t)

			if config.WithSetter {
				typeWrap += ".withSetter()"
			}

			return fmt.Sprintf(typeWrap, tp)
		},
	}

	tmpl := template.Must(
		template.New("mobxKeystoneModels").
			Funcs(funcs).
			ParseFS(
				templates,
				"templates/keystone-model.go.tmpl",
			),
	)

	mobxKeystoneImports := []string{"Model", "tProp", "types", "model"}

	for _, s := range st {
		for _, f := range s.Fields {
			if utils.ExistInStringArray([]string{"pgtype.JSONB", "map[string]interface{}"}, f.Type) &&
				!utils.ExistInStringArray(mobxKeystoneImports, "prop") {
				mobxKeystoneImports = append(mobxKeystoneImports, "prop")
				break
			}
		}
	}

	structs := []structParameters{}
	sortParam := strings.Split(config.Sort, ",")
	if len(sortParam) == 0 {
		for _, v := range st {
			structs = append(structs, v)
		}
	} else {
		for _, name := range sortParam {
			v, ok := st[name]
			if !ok {
				return nil, fmt.Errorf("sort error: undefined struct %s", name)
			}
			structs = append(structs, v)
		}

		keys := make([]string, 0, len(st))
		for k := range st {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			if utils.ExistInStringArray(sortParam, st[k].Name) {
				continue
			}
			structs = append(structs, st[k])
		}
	}

	tctx := tmplKeystoneCtx{
		Structs: structs,
		Imports: map[string][]string{
			"mobx-keystone": mobxKeystoneImports,
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

	cdata.data = b.Bytes()

	if config.PrettierCode {
		cdata.afterHook = func() error {
			fmt.Println("prettier generated typescript code ...")
			cmd := exec.Command("npx", "prettier", "--write", filepath.Join(cdata.outputDir, cdata.outputFileName))
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

func getKeystoneType(c config.GenModels, st Structs, sct Types, t string) (typeWrap string, tp string) {
	t = strings.ReplaceAll(t, "*", "")

	typeWrap = "tProp(types.maybe(%s))"
	typeWrapUnchecked := "prop<%s>()"

	tp = ""
	switch t {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		tp = "types.integer"
	case "float32", "float64":
		tp = "types.number"
	case "string":
		tp = "types.string"
	case "bool":
		tp = "types.boolean"
	case "bun.NullTime", "time.Time":
		tp = "types.dateString"
	case "map[string]interface{}", "pgtype.JSONB":
		tp = "Record<string, any>"
		typeWrap = typeWrapUnchecked
	default:
		_, okst := st[t]
		existScalarItem, oksct := sct[t]
		if okst {
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
