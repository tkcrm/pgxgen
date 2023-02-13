// Code generated by sqlc-pg-gen. DO NOT EDIT.

package contrib

import (
	"github.com/tkcrm/pgxgen/pkg/sqlc/sql/ast"
	"github.com/tkcrm/pgxgen/pkg/sqlc/sql/catalog"
)

var funcsFileFdw = []*catalog.Function{
	{
		Name:       "file_fdw_handler",
		Args:       []*catalog.Argument{},
		ReturnType: &ast.TypeName{Name: "fdw_handler"},
	},
	{
		Name: "file_fdw_validator",
		Args: []*catalog.Argument{
			{
				Type: &ast.TypeName{Name: "text[]"},
			},
			{
				Type: &ast.TypeName{Name: "oid"},
			},
		},
		ReturnType: &ast.TypeName{Name: "void"},
	},
}

func FileFdw() *catalog.Schema {
	s := &catalog.Schema{Name: "pg_catalog"}
	s.Funcs = funcsFileFdw
	return s
}
