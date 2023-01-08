package keystone

import "github.com/tkcrm/pgxgen/internal/structs"

type tmplKeystoneCtx struct {
	Structs                  structs.StructSlice
	Imports                  map[string][]string
	ImportTypes              map[string][]string
	DecoratorModelNamePrefix string
	ExportModelSuffix        string
	WithSetter               bool
	Version                  string
}
