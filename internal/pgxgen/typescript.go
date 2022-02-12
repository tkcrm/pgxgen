package pgxgen

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"unicode"

	"github.com/tkcrm/pgxgen/internal/config"
)

type tmplTypescriptCtx struct {
	Structs          StructSlice
	ExportTypePrefix string
	ExportTypeSuffix string
}

func generateTypescript(args []string, c config.Config) error {
	for _, config := range c.Pgxgen.GenTypescriptFromStructs {
		if config.OutputDir == "" {
			return fmt.Errorf("output dir is empty")
		}

		if config.OutputFileName == "" {
			return fmt.Errorf("output file name is empty")
		}

		dirItems, err := os.ReadDir(config.Path)
		if err != nil {
			return err
		}

		structs := make(Structs)

		for _, item := range dirItems {
			if item.IsDir() || !strings.HasSuffix(item.Name(), ".go") {
				continue
			}

			file, err := os.ReadFile(filepath.Join(config.Path, item.Name()))
			if err != nil {
				return err
			}

			for key, value := range getStructs(string(file)) {
				structs[key] = value
			}

		}

		// Filter structs
		for key, value := range structs {
			var deleteStruct *bool
			for _, restring := range config.IncludeStructNamesRegexp {
				r := regexp.MustCompile(restring)
				if !r.MatchString(value.Name) {
					if deleteStruct == nil {
						tr := true
						deleteStruct = &tr
					}
				} else {
					fs := false
					deleteStruct = &fs
				}
			}

			for _, restring := range config.ExcludeStructNamesRegexp {
				r := regexp.MustCompile(restring)
				if r.MatchString(value.Name) {
					tr := true
					deleteStruct = &tr
				}
			}

			if deleteStruct != nil && *deleteStruct {
				delete(structs, key)
			}
		}

		structsSlice := ConvertStructsToSlice(structs)
		if err := structsSlice.Sort(); err != nil {
			return err
		}

		typescriptCompiled, err := compileTypescript(config, structsSlice)
		if err != nil {
			return err
		}
		if err := compileTemplate(typescriptCompiled); err != nil {
			return err
		}

	}

	return nil
}

func compileTypescript(c config.GenTypescriptFromStructs, st StructSlice) (*compileData, error) {

	cdata := compileData{
		outputDir:      c.OutputDir,
		outputFileName: c.OutputFileName,
	}

	funcs := defaultTmplFuncs
	tmplAddFunc(funcs, "isEmptyFields", func(fields []*structField) bool {
		return len(fields) == 0
	})
	tmplAddFunc(funcs, "filterFields", func(fields []*structField) []*structField {
		resFields := make([]*structField, 0)
		for _, f := range fields {
			if unicode.IsLower(rune(f.Name[0])) {
				continue
			}
			resFields = append(resFields, f)
		}

		return resFields
	})
	tmplAddFunc(funcs, "isNullable", func(t string) bool {
		return strings.Contains(t, "*")
	})
	tmplAddFunc(funcs, "getType", func(t string) string {
		return getTypescriptType(st, t)
	})

	tmpl := template.Must(
		template.New("").
			Funcs(funcs).
			ParseFS(
				templates,
				"templates/typescript.go.tmpl",
			),
	)

	tctx := tmplTypescriptCtx{
		Structs:          st,
		ExportTypePrefix: c.ExportTypePrefix,
		ExportTypeSuffix: c.ExportTypeSuffix,
	}

	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	err := tmpl.ExecuteTemplate(w, "typescriptFile", &tctx)
	w.Flush()
	if err != nil {
		return nil, fmt.Errorf("execte template error: %s", err.Error())
	}

	cdata.data = b.Bytes()

	if c.PrettierCode {
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

func getTypescriptType(st StructSlice, t string) (tp string) {
	t = strings.ReplaceAll(t, "*", "")

	tp = ""
	switch t {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		tp = "number"
	case "float32", "float64":
		tp = "number"
	case "string":
		tp = "string"
	case "bool":
		tp = "boolean"
	case "bun.NullTime", "time.Time", "timestamppb.Timestamp":
		tp = "Date"
	case "map[string]interface{}", "pgtype.JSONB":
		tp = "Record<string, any>"
	default:
		tp = "any"
		fmt.Println("undefined type:", t)
	}
	return tp
}
