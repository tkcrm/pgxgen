// Code generated by sqlc-pg-gen. DO NOT EDIT.

package contrib

import (
	"github.com/tkcrm/pgxgen/pkg/sqlc/sql/ast"
	"github.com/tkcrm/pgxgen/pkg/sqlc/sql/catalog"
)

var funcsTcn = []*catalog.Function{
	{
		Name:       "triggered_change_notification",
		Args:       []*catalog.Argument{},
		ReturnType: &ast.TypeName{Name: "trigger"},
	},
}

func Tcn() *catalog.Schema {
	s := &catalog.Schema{Name: "pg_catalog"}
	s.Funcs = funcsTcn
	return s
}