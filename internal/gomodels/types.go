package gomodels

import "github.com/tkcrm/pgxgen/internal/structs"

type tmplGoModelsCtx struct {
	Version string
	Package string
	Structs structs.Structs
	Imports string
}
