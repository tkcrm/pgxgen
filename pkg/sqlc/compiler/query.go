package compiler

import (
	"github.com/tkcrm/pgxgen/pkg/sqlc/metadata"
	"github.com/tkcrm/pgxgen/pkg/sqlc/sql/ast"
	"github.com/tkcrm/pgxgen/pkg/sqlc/sql/catalog"
)

type Function struct {
	Rel        *ast.FuncName
	ReturnType *ast.TypeName
	Outs       []*catalog.Argument
}

type Table struct {
	Rel     *ast.TableName
	Columns []*Column
}

type Column struct {
	Name         string
	OriginalName string
	DataType     string
	NotNull      bool
	Unsigned     bool
	IsArray      bool
	ArrayDims    int
	Comment      string
	Length       *int
	IsNamedParam bool
	IsFuncCall   bool

	// XXX: Figure out what PostgreSQL calls `foo.id`
	Scope      string
	Table      *ast.TableName
	TableAlias string
	Type       *ast.TypeName
	EmbedTable *ast.TableName

	IsSqlcSlice bool // is this sqlc.slice()

	skipTableRequiredCheck bool
}

type Query struct {
	SQL      string
	Metadata metadata.Metadata
	Columns  []*Column
	Params   []Parameter

	// Needed for CopyFrom
	InsertIntoTable *ast.TableName

	// Needed for vet
	RawStmt *ast.RawStmt
}

type Parameter struct {
	Number int
	Column *Column
}
