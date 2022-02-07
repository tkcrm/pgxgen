package pgxgen

import "embed"

//go:embed templates/*
var templates embed.FS

type tmplCtx struct {
	Package string
	Structs Structs
	Imports string
}
