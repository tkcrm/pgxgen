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
			case *sqliteCreateIndex:
				p.handleCreateIndex(cat, n)
			case *sqliteDropIndex:
				p.handleDropIndex(cat, n)
			case *sqliteCreateView:
				p.handleCreateView(cat, n)
			case *sqliteDropView:
				p.handleDropView(cat, n)
			}
		}
	}

	return cat, nil
}

// sqliteStmt is a parsed SQLite statement.
type sqliteStmt any

type sqliteCreateTable struct {
	Schema      string
	Name        string
	IfNotExists bool
	Columns     []*catalog.Column
	PrimaryKey  *catalog.PrimaryKey
	ForeignKeys []*catalog.ForeignKey
	Uniques     []*catalog.UniqueConstraint
	Checks      []*catalog.CheckConstraint
}

type sqliteAlterTable struct {
	Schema      string
	Table       string
	AddColumn   *catalog.Column
	AddColumnFK *catalog.ForeignKey // FK from ADD COLUMN's REFERENCES
	DropColumn  string
	RenameTable string
}

type sqliteDropTable struct {
	Schema string
	Name   string
}

type sqliteCreateIndex struct {
	Schema    string
	TableName string
	Index     *catalog.Index
}

type sqliteDropIndex struct {
	Schema string
	Name   string
}

type sqliteCreateView struct {
	Schema      string
	Name        string
	IfNotExists bool
	ColumnNames []string
	Query       string // raw SELECT text
	SelectCore  *sqliteparser.Select_coreContext
}

type sqliteDropView struct {
	Schema string
	Name   string
}

type sqliteErrorListener struct {
	*antlr.DefaultErrorListener
	err string
}

func (el *sqliteErrorListener) SyntaxError(_ antlr.Recognizer, _ any, _, _ int, msg string, _ antlr.RecognitionException) {
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
			converted := convertSqliteStmts(stmt)
			stmts = append(stmts, converted...)
		}
	}

	return stmts, nil
}

func convertSqliteStmts(stmt sqliteparser.ISql_stmtContext) []sqliteStmt {
	s, ok := stmt.(*sqliteparser.Sql_stmtContext)
	if !ok {
		return nil
	}

	if ct := s.Create_table_stmt(); ct != nil {
		return []sqliteStmt{convertSqliteCreateTable(ct)}
	}
	if at := s.Alter_table_stmt(); at != nil {
		return []sqliteStmt{convertSqliteAlterTable(at)}
	}
	if dt := s.Drop_stmt(); dt != nil {
		if r := convertSqliteDropStmt(dt); r != nil {
			return []sqliteStmt{r}
		}
	}
	if ci := s.Create_index_stmt(); ci != nil {
		return []sqliteStmt{convertSqliteCreateIndexStmt(ci)}
	}
	if cv := s.Create_view_stmt(); cv != nil {
		return []sqliteStmt{convertSqliteCreateView(cv)}
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
			Name:     sqliteIdentifier(def.Column_name().GetText()),
			Type:     typeName,
			FullType: typeName,
		}

		// Process column constraints
		for _, ic := range def.AllColumn_constraint() {
			constraint, ok := ic.(*sqliteparser.Column_constraintContext)
			if !ok {
				continue
			}
			sqliteProcessColumnConstraint(col, constraint, ct)
		}

		ct.Columns = append(ct.Columns, col)
	}

	// Process table constraints
	for _, itc := range n.AllTable_constraint() {
		tc, ok := itc.(*sqliteparser.Table_constraintContext)
		if !ok {
			continue
		}
		sqliteProcessTableConstraint(tc, ct)
	}

	// Build PrimaryKey from inline column-level PK if no table-level PK was set
	if ct.PrimaryKey == nil {
		var pkCols []string
		for _, col := range ct.Columns {
			if col.IsPrimary {
				pkCols = append(pkCols, col.Name)
			}
		}
		if len(pkCols) > 0 {
			ct.PrimaryKey = &catalog.PrimaryKey{Columns: pkCols}
		}
	}

	return ct
}

// sqliteProcessColumnConstraint extracts constraint info from a column-level constraint.
func sqliteProcessColumnConstraint(col *catalog.Column, c *sqliteparser.Column_constraintContext, ct *sqliteCreateTable) {
	if c.PRIMARY_() != nil && c.KEY_() != nil {
		col.IsPrimary = true
		col.NotNull = true
		return
	}
	if c.NOT_() != nil && c.NULL_() != nil {
		col.NotNull = true
		return
	}
	if c.DEFAULT_() != nil {
		if sn := c.Signed_number(); sn != nil {
			col.Default = sn.GetText()
		} else if lv := c.Literal_value(); lv != nil {
			col.Default = lv.GetText()
		} else if expr := c.Expr(); expr != nil {
			col.Default = expr.GetText()
		}
		return
	}
	if c.UNIQUE_() != nil {
		name := ""
		if c.CONSTRAINT_() != nil && c.Name() != nil {
			name = sqliteIdentifier(c.Name().GetText())
		}
		ct.Uniques = append(ct.Uniques, &catalog.UniqueConstraint{
			Name:    name,
			Columns: []string{col.Name},
		})
		return
	}
	if c.CHECK_() != nil {
		if expr := c.Expr(); expr != nil {
			name := ""
			if c.CONSTRAINT_() != nil && c.Name() != nil {
				name = sqliteIdentifier(c.Name().GetText())
			}
			ct.Checks = append(ct.Checks, &catalog.CheckConstraint{
				Name:       name,
				Expression: expr.GetText(),
			})
		}
		return
	}
	if fkc := c.Foreign_key_clause(); fkc != nil {
		fk := sqliteParseForeignKeyClause(fkc)
		if fk != nil {
			fk.Columns = []string{col.Name}
			ct.ForeignKeys = append(ct.ForeignKeys, fk)
		}
	}
}

// sqliteProcessTableConstraint extracts constraint info from a table-level constraint.
func sqliteProcessTableConstraint(tc *sqliteparser.Table_constraintContext, ct *sqliteCreateTable) {
	constraintName := ""
	if tc.CONSTRAINT_() != nil && tc.Name() != nil {
		constraintName = sqliteIdentifier(tc.Name().GetText())
	}

	if tc.PRIMARY_() != nil && tc.KEY_() != nil {
		pk := &catalog.PrimaryKey{Name: constraintName}
		for _, ic := range tc.AllIndexed_column() {
			if idx, ok := ic.(*sqliteparser.Indexed_columnContext); ok {
				if cn := idx.Column_name(); cn != nil {
					pk.Columns = append(pk.Columns, sqliteIdentifier(cn.GetText()))
				} else if expr := idx.Expr(); expr != nil {
					pk.Columns = append(pk.Columns, expr.GetText())
				}
			}
		}
		ct.PrimaryKey = pk
		// Mark columns as NOT NULL and IsPrimary
		for _, pkCol := range pk.Columns {
			for _, col := range ct.Columns {
				if col.Name == pkCol {
					col.NotNull = true
					col.IsPrimary = true
				}
			}
		}
		return
	}

	if tc.UNIQUE_() != nil {
		uc := &catalog.UniqueConstraint{Name: constraintName}
		for _, ic := range tc.AllIndexed_column() {
			if idx, ok := ic.(*sqliteparser.Indexed_columnContext); ok {
				if cn := idx.Column_name(); cn != nil {
					uc.Columns = append(uc.Columns, sqliteIdentifier(cn.GetText()))
				}
			}
		}
		ct.Uniques = append(ct.Uniques, uc)
		return
	}

	if tc.CHECK_() != nil {
		if expr := tc.Expr(); expr != nil {
			ct.Checks = append(ct.Checks, &catalog.CheckConstraint{
				Name:       constraintName,
				Expression: expr.GetText(),
			})
		}
		return
	}

	if tc.FOREIGN_() != nil && tc.KEY_() != nil {
		fk := sqliteParseForeignKeyClause(tc.Foreign_key_clause())
		if fk != nil {
			fk.Name = constraintName
			for _, cn := range tc.AllColumn_name() {
				fk.Columns = append(fk.Columns, sqliteIdentifier(cn.GetText()))
			}
			ct.ForeignKeys = append(ct.ForeignKeys, fk)
		}
	}
}

// sqliteParseForeignKeyClause extracts FK details from a foreign_key_clause ANTLR node.
func sqliteParseForeignKeyClause(ctx sqliteparser.IForeign_key_clauseContext) *catalog.ForeignKey {
	fkc, ok := ctx.(*sqliteparser.Foreign_key_clauseContext)
	if !ok || fkc == nil {
		return nil
	}

	fk := &catalog.ForeignKey{}
	if ft := fkc.Foreign_table(); ft != nil {
		fk.RefTable = sqliteIdentifier(ft.GetText())
	}
	for _, cn := range fkc.AllColumn_name() {
		fk.RefColumns = append(fk.RefColumns, sqliteIdentifier(cn.GetText()))
	}

	// Parse ON DELETE / ON UPDATE actions by walking children
	children := fkc.GetChildren()
	for i := range children {
		tn, ok := children[i].(antlr.TerminalNode)
		if !ok {
			continue
		}
		if tn.GetSymbol().GetTokenType() != sqliteparser.SQLiteParserON_ {
			continue
		}
		// Next token should be DELETE_ or UPDATE_
		if i+1 >= len(children) {
			continue
		}
		nextTn, ok := children[i+1].(antlr.TerminalNode)
		if !ok {
			continue
		}
		isDelete := nextTn.GetSymbol().GetTokenType() == sqliteparser.SQLiteParserDELETE_
		isUpdate := nextTn.GetSymbol().GetTokenType() == sqliteparser.SQLiteParserUPDATE_
		if !isDelete && !isUpdate {
			continue
		}
		// Next token(s) should be the action
		action := sqliteParseAction(children, i+2)
		if isDelete {
			fk.OnDelete = action
		} else {
			fk.OnUpdate = action
		}
	}

	return fk
}

// sqliteParseAction extracts the action keyword(s) starting from index i in children.
func sqliteParseAction(children []antlr.Tree, i int) string {
	if i >= len(children) {
		return ""
	}
	tn, ok := children[i].(antlr.TerminalNode)
	if !ok {
		return ""
	}
	switch tn.GetSymbol().GetTokenType() {
	case sqliteparser.SQLiteParserCASCADE_:
		return "CASCADE"
	case sqliteparser.SQLiteParserRESTRICT_:
		return "RESTRICT"
	case sqliteparser.SQLiteParserSET_:
		if i+1 < len(children) {
			if next, ok := children[i+1].(antlr.TerminalNode); ok {
				switch next.GetSymbol().GetTokenType() {
				case sqliteparser.SQLiteParserNULL_:
					return "SET NULL"
				case sqliteparser.SQLiteParserDEFAULT_:
					return "SET DEFAULT"
				}
			}
		}
	case sqliteparser.SQLiteParserNO_:
		if i+1 < len(children) {
			if next, ok := children[i+1].(antlr.TerminalNode); ok {
				if next.GetSymbol().GetTokenType() == sqliteparser.SQLiteParserACTION_ {
					return "NO ACTION"
				}
			}
		}
	}
	return ""
}

func convertSqliteCreateIndexStmt(ctx sqliteparser.ICreate_index_stmtContext) *sqliteCreateIndex {
	n, ok := ctx.(*sqliteparser.Create_index_stmtContext)
	if !ok {
		return nil
	}

	ci := &sqliteCreateIndex{}
	if n.Schema_name() != nil {
		ci.Schema = n.Schema_name().GetText()
	}
	ci.TableName = sqliteIdentifier(n.Table_name().GetText())

	idx := &catalog.Index{
		IsUnique:    n.UNIQUE_() != nil,
		IfNotExists: n.EXISTS_() != nil,
	}
	if n.Index_name() != nil {
		idx.Name = sqliteIdentifier(n.Index_name().GetText())
	}
	for _, ic := range n.AllIndexed_column() {
		if idxCol, ok := ic.(*sqliteparser.Indexed_columnContext); ok {
			if cn := idxCol.Column_name(); cn != nil {
				idx.Columns = append(idx.Columns, sqliteIdentifier(cn.GetText()))
			} else if expr := idxCol.Expr(); expr != nil {
				idx.Columns = append(idx.Columns, expr.GetText())
			}
		}
	}
	if n.WHERE_() != nil && n.Expr() != nil {
		idx.Where = n.Expr().GetText()
	}

	ci.Index = idx
	return ci
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
			col := &catalog.Column{
				Name:     sqliteIdentifier(def.Column_name().GetText()),
				Type:     typeName,
				FullType: typeName,
			}

			// Process column constraints (including DEFAULT, FK, etc.)
			tempCT := &sqliteCreateTable{} // temporary holder for FK/unique/check
			for _, ic := range def.AllColumn_constraint() {
				constraint, ok := ic.(*sqliteparser.Column_constraintContext)
				if !ok {
					continue
				}
				sqliteProcessColumnConstraint(col, constraint, tempCT)
			}
			at.AddColumn = col
			// If a FK was found in column constraints, pass it along
			if len(tempCT.ForeignKeys) > 0 {
				at.AddColumnFK = tempCT.ForeignKeys[0]
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

func convertSqliteDropStmt(ctx sqliteparser.IDrop_stmtContext) sqliteStmt {
	n, ok := ctx.(*sqliteparser.Drop_stmtContext)
	if !ok {
		return nil
	}

	// DROP TABLE
	if n.TABLE_() != nil {
		dt := &sqliteDropTable{}
		if n.Schema_name() != nil {
			dt.Schema = n.Schema_name().GetText()
		}
		if n.Any_name() != nil {
			dt.Name = sqliteIdentifier(n.Any_name().GetText())
		}
		return dt
	}

	// DROP VIEW
	if n.VIEW_() != nil {
		dv := &sqliteDropView{}
		if n.Schema_name() != nil {
			dv.Schema = n.Schema_name().GetText()
		}
		if n.Any_name() != nil {
			dv.Name = sqliteIdentifier(n.Any_name().GetText())
		}
		return dv
	}

	// DROP INDEX
	if n.INDEX_() != nil {
		di := &sqliteDropIndex{}
		if n.Schema_name() != nil {
			di.Schema = n.Schema_name().GetText()
		}
		if n.Any_name() != nil {
			di.Name = sqliteIdentifier(n.Any_name().GetText())
		}
		return di
	}

	return nil
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
		Name:        ct.Name,
		Schema:      schemaName,
		Columns:     ct.Columns,
		PrimaryKey:  ct.PrimaryKey,
		ForeignKeys: ct.ForeignKeys,
		Uniques:     ct.Uniques,
		Checks:      ct.Checks,
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
		if at.AddColumnFK != nil {
			table.ForeignKeys = append(table.ForeignKeys, at.AddColumnFK)
		}
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

func (p *sqliteParser) handleCreateIndex(cat *catalog.Catalog, ci *sqliteCreateIndex) {
	schemaName := ci.Schema
	if schemaName == "" {
		schemaName = cat.DefaultSchema
	}

	table := p.findTable(cat, schemaName, ci.TableName)
	if table == nil {
		return
	}

	table.Indexes = append(table.Indexes, ci.Index)
}

func (p *sqliteParser) handleDropIndex(cat *catalog.Catalog, di *sqliteDropIndex) {
	schemaName := di.Schema
	if schemaName == "" {
		schemaName = cat.DefaultSchema
	}

	for _, s := range cat.Schemas {
		if s.Name == schemaName {
			for _, t := range s.Tables {
				for i, idx := range t.Indexes {
					if strings.EqualFold(idx.Name, di.Name) {
						t.Indexes = append(t.Indexes[:i], t.Indexes[i+1:]...)
						return
					}
				}
			}
		}
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

func convertSqliteCreateView(ctx sqliteparser.ICreate_view_stmtContext) *sqliteCreateView {
	n, ok := ctx.(*sqliteparser.Create_view_stmtContext)
	if !ok {
		return nil
	}

	cv := &sqliteCreateView{
		IfNotExists: n.EXISTS_() != nil,
	}

	if n.Schema_name() != nil {
		cv.Schema = n.Schema_name().GetText()
	}
	if n.View_name() != nil {
		cv.Name = sqliteIdentifier(n.View_name().GetText())
	}

	// Explicit column names
	for _, cn := range n.AllColumn_name() {
		cv.ColumnNames = append(cv.ColumnNames, sqliteIdentifier(cn.GetText()))
	}

	// Extract raw SELECT text from original token stream
	if selStmt := n.Select_stmt(); selStmt != nil {
		selCtx := selStmt.(*sqliteparser.Select_stmtContext)
		start := selCtx.GetStart()
		stop := selCtx.GetStop()
		if start != nil && stop != nil {
			stream := start.GetTokenSource().GetInputStream()
			cv.Query = stream.GetText(start.GetStart(), stop.GetStop())
		}
		// Capture Select_core for column resolution
		if cores := selCtx.AllSelect_core(); len(cores) > 0 {
			if sc, ok := cores[0].(*sqliteparser.Select_coreContext); ok {
				cv.SelectCore = sc
			}
		}
	}

	return cv
}

func (p *sqliteParser) handleCreateView(cat *catalog.Catalog, cv *sqliteCreateView) {
	schemaName := cv.Schema
	if schemaName == "" {
		schemaName = cat.DefaultSchema
	}

	schema := p.getOrCreateSchema(cat, schemaName)

	if cv.IfNotExists {
		for _, v := range schema.Views {
			if strings.EqualFold(v.Name, cv.Name) {
				return
			}
		}
	}

	// Resolve columns from the query by looking up referenced tables
	columns := p.resolveViewColumns(cat, cv)

	schema.Views = append(schema.Views, &catalog.View{
		Name:    cv.Name,
		Schema:  schemaName,
		Columns: columns,
		Query:   cv.Query,
	})
}

func (p *sqliteParser) resolveViewColumns(cat *catalog.Catalog, cv *sqliteCreateView) []*catalog.Column {
	// If explicit column names are provided, use them with fallback type "any"
	if len(cv.ColumnNames) > 0 {
		var columns []*catalog.Column
		for _, name := range cv.ColumnNames {
			columns = append(columns, &catalog.Column{
				Name:     name,
				Type:     "any",
				FullType: "any",
			})
		}
		return columns
	}

	// Use the captured SelectCore from the ANTLR parse tree
	if cv.SelectCore == nil {
		return nil
	}

	// Extract FROM tables for type resolution
	fromTables := p.extractFromTablesSQL(cv.SelectCore)

	var columns []*catalog.Column
	for _, irc := range cv.SelectCore.AllResult_column() {
		rc, ok := irc.(*sqliteparser.Result_columnContext)
		if !ok {
			continue
		}

		// Handle SELECT *
		if rc.STAR() != nil {
			tableName := ""
			if rc.Table_name() != nil {
				tableName = sqliteIdentifier(rc.Table_name().GetText())
			}
			expanded := p.expandSqliteStar(cat, tableName, fromTables)
			columns = append(columns, expanded...)
			continue
		}

		colName := ""
		colType := "any"

		// Get alias
		if rc.Column_alias() != nil {
			colName = sqliteIdentifier(rc.Column_alias().GetText())
		}

		// Try to extract column reference from expression
		if rc.Expr() != nil {
			text := rc.Expr().GetText()
			// Simple column reference: "table.column" or "column"
			if colName == "" {
				parts := strings.Split(text, ".")
				colName = sqliteIdentifier(parts[len(parts)-1])
			}
			// Try to resolve type
			parts := strings.Split(text, ".")
			var refTable, refCol string
			if len(parts) == 2 {
				refTable = sqliteIdentifier(parts[0])
				refCol = sqliteIdentifier(parts[1])
			} else if len(parts) == 1 {
				refCol = sqliteIdentifier(parts[0])
			}
			if resolved := p.findSqliteColumnType(cat, refTable, refCol, fromTables); resolved != "" {
				colType = resolved
			}
		}

		if colName == "" {
			colName = fmt.Sprintf("column%d", len(columns)+1)
		}

		columns = append(columns, &catalog.Column{
			Name:     colName,
			Type:     colType,
			FullType: colType,
		})
	}

	return columns
}

func (p *sqliteParser) extractFromTablesSQL(sc *sqliteparser.Select_coreContext) map[string]string {
	tables := make(map[string]string)
	for _, itos := range sc.AllTable_or_subquery() {
		tos, ok := itos.(*sqliteparser.Table_or_subqueryContext)
		if !ok || tos.Table_name() == nil {
			continue
		}
		name := sqliteIdentifier(tos.Table_name().GetText())
		alias := name
		if tos.Table_alias() != nil {
			alias = sqliteIdentifier(tos.Table_alias().GetText())
		}
		tables[alias] = name
	}
	// Also check join clause
	if jc := sc.Join_clause(); jc != nil {
		if joinCtx, ok := jc.(*sqliteparser.Join_clauseContext); ok {
			for _, itos := range joinCtx.AllTable_or_subquery() {
				tos, ok := itos.(*sqliteparser.Table_or_subqueryContext)
				if !ok || tos.Table_name() == nil {
					continue
				}
				name := sqliteIdentifier(tos.Table_name().GetText())
				alias := name
				if tos.Table_alias() != nil {
					alias = sqliteIdentifier(tos.Table_alias().GetText())
				}
				tables[alias] = name
			}
		}
	}
	return tables
}

func (p *sqliteParser) expandSqliteStar(cat *catalog.Catalog, tableName string, fromTables map[string]string) []*catalog.Column {
	var columns []*catalog.Column

	if tableName != "" {
		realName := tableName
		if real, ok := fromTables[tableName]; ok {
			realName = real
		}
		if cols := p.findTableOrViewColumns(cat, realName); cols != nil {
			for _, col := range cols {
				columns = append(columns, &catalog.Column{
					Name:     col.Name,
					Type:     col.Type,
					FullType: col.FullType,
					NotNull:  col.NotNull,
				})
			}
		}
		return columns
	}

	for _, realName := range fromTables {
		if cols := p.findTableOrViewColumns(cat, realName); cols != nil {
			for _, col := range cols {
				columns = append(columns, &catalog.Column{
					Name:     col.Name,
					Type:     col.Type,
					FullType: col.FullType,
					NotNull:  col.NotNull,
				})
			}
		}
	}
	return columns
}

func (p *sqliteParser) findTableOrViewColumns(cat *catalog.Catalog, name string) []*catalog.Column {
	for _, s := range cat.Schemas {
		for _, t := range s.Tables {
			if strings.EqualFold(t.Name, name) {
				return t.Columns
			}
		}
		for _, v := range s.Views {
			if strings.EqualFold(v.Name, name) {
				return v.Columns
			}
		}
	}
	return nil
}

func (p *sqliteParser) findSqliteColumnType(cat *catalog.Catalog, tableName, colName string, fromTables map[string]string) string {
	if colName == "" {
		return ""
	}

	if tableName != "" {
		if real, ok := fromTables[tableName]; ok {
			tableName = real
		}
	}

	for _, s := range cat.Schemas {
		for _, t := range s.Tables {
			if tableName != "" && !strings.EqualFold(t.Name, tableName) {
				continue
			}
			for _, c := range t.Columns {
				if strings.EqualFold(c.Name, colName) {
					return c.Type
				}
			}
		}
		for _, v := range s.Views {
			if tableName != "" && !strings.EqualFold(v.Name, tableName) {
				continue
			}
			for _, c := range v.Columns {
				if strings.EqualFold(c.Name, colName) {
					return c.Type
				}
			}
		}
	}
	return ""
}

func (p *sqliteParser) handleDropView(cat *catalog.Catalog, dv *sqliteDropView) {
	schemaName := dv.Schema
	if schemaName == "" {
		schemaName = cat.DefaultSchema
	}

	for _, s := range cat.Schemas {
		if s.Name == schemaName {
			for i, v := range s.Views {
				if strings.EqualFold(v.Name, dv.Name) {
					s.Views = append(s.Views[:i], s.Views[i+1:]...)
					return
				}
			}
		}
	}
}

func sqliteIdentifier(id string) string {
	if len(id) >= 2 && id[0] == '"' && id[len(id)-1] == '"' {
		unquoted, _ := strconv.Unquote(id)
		return unquoted
	}
	return strings.ToLower(id)
}
