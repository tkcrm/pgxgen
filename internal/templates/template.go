package templates

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/gobeam/stringy"
)

//go:embed templates/*
var Templates embed.FS

func TmplAddFunc(funcs template.FuncMap, key string, value interface{}) {
	funcs[key] = value
}

var DefaultTmplFuncs = template.FuncMap{
	"lower": strings.ToLower,
	"snake_case": func(str string) string {
		return stringy.New(str).SnakeCase().ToLower()
	},
	"camel_case": func(str string) string {
		return stringy.New(str).CamelCase()
	},
	"lcfirst": func(str string) string {
		if str == "ID" {
			return "id"
		}
		return stringy.New(str).LcFirst()
	},
	"replace_id": func(str string) string {
		return strings.ReplaceAll(str, "ID", "Id")
	},
	"join": strings.Join,
}

type CompileData struct {
	Data           []byte
	OutputDir      string
	OutputFileName string
	AfterHook      func() error
}

func CompileTemplate(d *CompileData) error {
	if d.Data == nil && len(d.Data) == 0 {
		return fmt.Errorf("compile template error: data is undefined")
	}

	if d.OutputDir == "" {
		return fmt.Errorf("compile template error: output dir is empty")
	}

	if d.OutputFileName == "" {
		return fmt.Errorf("compile template error: output file name is empty")
	}

	if err := os.MkdirAll(d.OutputDir, os.ModePerm); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(d.OutputDir, d.OutputFileName), d.Data, os.ModePerm); err != nil {
		return fmt.Errorf("write error: %s", err.Error())
	}

	fmt.Println("successfully generated in:", filepath.Join(d.OutputDir, d.OutputFileName))

	if d.AfterHook != nil {
		if err := d.AfterHook(); err != nil {
			return err
		}
	}

	return nil
}
