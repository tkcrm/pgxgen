package pgxgen

import (
	"embed"
	"strings"
	"text/template"

	"github.com/gobeam/stringy"
)

//go:embed templates/*
var templates embed.FS

func tmplAddFunc(funcs template.FuncMap, key string, value interface{}) {
	funcs[key] = value
}

var defaultTmplFuncs = template.FuncMap{
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
}
