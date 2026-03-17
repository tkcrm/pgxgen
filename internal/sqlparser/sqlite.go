package sqlparser

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/tkcrm/pgxgen/internal/sqlparser/catalog"
	"github.com/tkcrm/pgxgen/internal/sqlparser/sqliteparser"
)

type sqliteParser struct{}

func newSqliteParser() *sqliteParser {
	return &sqliteParser{}
}

func (p *sqliteParser) ParseSchema(files []string) (*catalog.Catalog, error) {
	cat := &catalog.Catalog{
		DefaultSchema: "main",
		Schemas: []*catalog.Schema{
			{Name: "main"},
		},
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", file, err)
		}

		stmts, err := sqliteParse(string(data))
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", file, err)
		}

		for _, stmt := range stmts {
			switch n := stmt.(type) {
			case *sqliteCreateTable:
				p.handleCreateTable(cat, n)
			case *sqliteAlterTable:
				p.handleAlterTable(cat, n)
			case *sqliteDropTable:
				p.handleDropTable(cat, n)
			}
		}
	}

	return cat, nil
}

// sqliteStmt is a parsed SQLite statement.
type sqliteStmt interface{}

type sqliteCreateTable struct {
	Schema      string
	Name        string
	IfNotExists bool
	Columns     []*catalog.Column
}

type sqliteAlterTable struct {
	Schema      string
	Table       string
	AddColumn   *catalog.Column
	DropColumn  string
	RenameTable string
}

type sqliteDropTable struct {
	Schema string
	Name   string
}

type sqliteErrorListener struct {
	*antlr.DefaultErrorListener
	err string
}

func (el *sqliteErrorListener) SyntaxError(_ antlr.Recognizer, _ interface{}, _, _ int, msg string, _ antlr.RecognitionException) {
	el.err = msg
}

func sqliteParse(sql string) ([]sqliteStmt, error) {
	input := antlr.NewInputStream(sql)
	lexer := sqliteparser.NewSQLiteLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, 0)
	pp := sqliteparser.NewSQLiteParser(stream)
	el := &sqliteErrorListener{}
	pp.AddErrorListener(el)
	tree := pp.Parse()
	if el.err != "" {
		return nil, errors.New(el.err)
	}

	pctx, ok := tree.(*sqliteparser.ParseContext)
	if !ok {
		return nil, fmt.Errorf("expected ParseContext; got %T", tree)
	}

	var stmts []sqliteStmt
	for _, istmt := range pctx.AllSql_stmt_list() {
		list, ok := istmt.(*sqliteparser.Sql_stmt_listContext)
		if !ok {
			continue
		}
		for _, stmt := range list.AllSql_stmt() {
			s := convertSqliteStmt(stmt)
			if s != nil {
				stmts = append(stmts, s)
			}
		}
	}

	return stmts, nil
}

func convertSqliteStmt(stmt sqliteparser.ISql_stmtContext) sqliteStmt {
	s, ok := stmt.(*sqliteparser.Sql_stmtContext)
	if !ok {
		return nil
	}

	if ct := s.Create_table_stmt(); ct != nil {
		return convertSqliteCreateTable(ct)
	}
	if at := s.Alter_table_stmt(); at != nil {
		return convertSqliteAlterTable(at)
	}
	if dt := s.Drop_stmt(); dt != nil {
		return convertSqliteDropStmt(dt)
	}

	return nil
}

func convertSqliteCreateTable(ctx sqliteparser.ICreate_table_stmtContext) *sqliteCreateTable {
	n, ok := ctx.(*sqliteparser.Create_table_stmtContext)
	if !ok {
		return nil
	}

	ct := &sqliteCreateTable{
		IfNotExists: n.EXISTS_() != nil,
	}

	if n.Schema_name() != nil {
		ct.Schema = n.Schema_name().GetText()
	}
	ct.Name = sqliteIdentifier(n.Table_name().GetText())

	for _, idef := range n.AllColumn_def() {
		def, ok := idef.(*sqliteparser.Column_defContext)
		if !ok {
			continue
		}

		typeName := "any"
		if def.Type_name() != nil {
			typeName = def.Type_name().GetText()
		}

		col := &catalog.Column{
			Name:    sqliteIdentifier(def.Column_name().GetText()),
			Type:    typeName,
			NotNull: sqliteHasNotNullConstraint(def.AllColumn_constraint()),
		}

		ct.Columns = append(ct.Columns, col)
	}

	return ct
}

func convertSqliteAlterTable(ctx sqliteparser.IAlter_table_stmtContext) *sqliteAlterTable {
	n, ok := ctx.(*sqliteparser.Alter_table_stmtContext)
	if !ok {
		return nil
	}

	at := &sqliteAlterTable{}

	if n.Schema_name() != nil {
		at.Schema = n.Schema_name().GetText()
	}
	if n.Table_name() != nil {
		at.Table = sqliteIdentifier(n.Table_name().GetText())
	}

	// ADD COLUMN
	if n.ADD_() != nil {
		if def, ok := n.Column_def().(*sqliteparser.Column_defContext); ok {
			typeName := "any"
			if def.Type_name() != nil {
				typeName = def.Type_name().GetText()
			}
			at.AddColumn = &catalog.Column{
				Name:    sqliteIdentifier(def.Column_name().GetText()),
				Type:    typeName,
				NotNull: sqliteHasNotNullConstraint(def.AllColumn_constraint()),
			}
		}
	}

	// DROP COLUMN
	if n.DROP_() != nil && n.COLUMN_() != nil {
		cols := n.AllColumn_name()
		if len(cols) > 0 {
			at.DropColumn = sqliteIdentifier(cols[0].GetText())
		}
	}

	// RENAME TO
	if n.RENAME_() != nil && n.New_table_name() != nil {
		at.RenameTable = sqliteIdentifier(n.New_table_name().GetText())
	}

	return at
}

func convertSqliteDropStmt(ctx sqliteparser.IDrop_stmtContext) *sqliteDropTable {
	n, ok := ctx.(*sqliteparser.Drop_stmtContext)
	if !ok {
		return nil
	}

	// Only handle DROP TABLE
	if n.TABLE_() == nil {
		return nil
	}

	dt := &sqliteDropTable{}
	if n.Schema_name() != nil {
		dt.Schema = n.Schema_name().GetText()
	}
	if n.Any_name() != nil {
		dt.Name = sqliteIdentifier(n.Any_name().GetText())
	}
	return dt
}

func (p *sqliteParser) handleCreateTable(cat *catalog.Catalog, ct *sqliteCreateTable) {
	schemaName := ct.Schema
	if schemaName == "" {
		schemaName = cat.DefaultSchema
	}

	schema := p.getOrCreateSchema(cat, schemaName)

	if ct.IfNotExists {
		for _, t := range schema.Tables {
			if t.Name == ct.Name {
				return
			}
		}
	}

	table := &catalog.Table{
		Name:    ct.Name,
		Schema:  schemaName,
		Columns: ct.Columns,
	}

	schema.Tables = append(schema.Tables, table)
}

func (p *sqliteParser) handleAlterTable(cat *catalog.Catalog, at *sqliteAlterTable) {
	schemaName := at.Schema
	if schemaName == "" {
		schemaName = cat.DefaultSchema
	}

	table := p.findTable(cat, schemaName, at.Table)
	if table == nil {
		return
	}

	if at.AddColumn != nil {
		table.Columns = append(table.Columns, at.AddColumn)
	}

	if at.DropColumn != "" {
		for i, c := range table.Columns {
			if c.Name == at.DropColumn {
				table.Columns = append(table.Columns[:i], table.Columns[i+1:]...)
				break
			}
		}
	}

	if at.RenameTable != "" {
		table.Name = at.RenameTable
	}
}

func (p *sqliteParser) findTable(cat *catalog.Catalog, schemaName, tableName string) *catalog.Table {
	for _, s := range cat.Schemas {
		if s.Name == schemaName {
			for _, t := range s.Tables {
				if strings.EqualFold(t.Name, tableName) {
					return t
				}
			}
		}
	}
	return nil
}

func (p *sqliteParser) handleDropTable(cat *catalog.Catalog, dt *sqliteDropTable) {
	schemaName := dt.Schema
	if schemaName == "" {
		schemaName = cat.DefaultSchema
	}

	for _, s := range cat.Schemas {
		if s.Name == schemaName {
			for i, t := range s.Tables {
				if strings.EqualFold(t.Name, dt.Name) {
					s.Tables = append(s.Tables[:i], s.Tables[i+1:]...)
					return
				}
			}
		}
	}
}

func (p *sqliteParser) getOrCreateSchema(cat *catalog.Catalog, name string) *catalog.Schema {
	for _, s := range cat.Schemas {
		if s.Name == name {
			return s
		}
	}
	s := &catalog.Schema{Name: name}
	cat.Schemas = append(cat.Schemas, s)
	return s
}

func sqliteIdentifier(id string) string {
	if len(id) >= 2 && id[0] == '"' && id[len(id)-1] == '"' {
		unquoted, _ := strconv.Unquote(id)
		return unquoted
	}
	return strings.ToLower(id)
}

func sqliteHasNotNullConstraint(checks []sqliteparser.IColumn_constraintContext) bool {
	for _, c := range checks {
		constraint, ok := c.(*sqliteparser.Column_constraintContext)
		if !ok {
			continue
		}
		if constraint.PRIMARY_() != nil && constraint.KEY_() != nil {
			return true
		}
		if constraint.NOT_() != nil && constraint.NULL_() != nil {
			return true
		}
	}
	return false
}
