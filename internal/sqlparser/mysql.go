package sqlparser

import (
	"fmt"
	"os"
	"strings"

	pcast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/format"
	"github.com/pingcap/tidb/pkg/parser/mysql"
	"github.com/pingcap/tidb/pkg/parser/types"
	"github.com/tkcrm/pgxgen/internal/sqlparser/catalog"

	tidbparser "github.com/pingcap/tidb/pkg/parser"
	_ "github.com/pingcap/tidb/pkg/parser/test_driver"
)

type mysqlParser struct{}

func newMysqlParser() *mysqlParser {
	return &mysqlParser{}
}

func (p *mysqlParser) ParseSchema(files []string) (*catalog.Catalog, error) {
	cat := &catalog.Catalog{
		DefaultSchema: "public",
		Schemas: []*catalog.Schema{
			{Name: "public"},
		},
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", file, err)
		}

		parser := tidbparser.New()
		stmts, _, err := parser.Parse(string(data), "", "")
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", file, err)
		}

		for _, stmt := range stmts {
			switch n := stmt.(type) {
			case *pcast.CreateTableStmt:
				p.handleCreateTable(cat, n)
			case *pcast.AlterTableStmt:
				p.handleAlterTable(cat, n)
			case *pcast.DropTableStmt:
				if n.IsView {
					p.handleDropView(cat, n)
				} else {
					p.handleDropTable(cat, n)
				}
			case *pcast.CreateViewStmt:
				p.handleCreateView(cat, n)
			}
		}
	}

	return cat, nil
}

func (p *mysqlParser) handleCreateTable(cat *catalog.Catalog, n *pcast.CreateTableStmt) {
	schemaName := n.Table.Schema.String()
	if schemaName == "" {
		schemaName = cat.DefaultSchema
	}
	tableName := n.Table.Name.String()

	schema := p.getOrCreateSchema(cat, schemaName)

	// Check if table already exists (IF NOT EXISTS)
	if n.IfNotExists {
		for _, t := range schema.Tables {
			if t.Name == tableName {
				return
			}
		}
	}

	table := &catalog.Table{
		Name:   tableName,
		Schema: schemaName,
	}

	// Extract table comment
	for _, opt := range n.Options {
		if opt.Tp == pcast.TableOptionComment {
			table.Comment = opt.StrValue
		}
	}

	for _, def := range n.Cols {
		col := p.convertColumnDef(def)
		if col != nil {
			table.Columns = append(table.Columns, col)
		}
	}

	schema.Tables = append(schema.Tables, table)
}

func (p *mysqlParser) convertColumnDef(def *pcast.ColumnDef) *catalog.Column {
	col := &catalog.Column{
		Name:       def.Name.String(),
		Type:       types.TypeToStr(def.Tp.GetType(), def.Tp.GetCharset()),
		NotNull:    mysqlIsNotNull(def),
		IsUnsigned: mysql.HasUnsignedFlag(def.Tp.GetFlag()),
	}

	if def.Tp.GetFlen() >= 0 {
		length := def.Tp.GetFlen()
		col.Length = &length
	}

	// Extract column comment
	for _, opt := range def.Options {
		if opt.Tp == pcast.ColumnOptionComment {
			if value, ok := opt.Expr.(interface{ GetString() string }); ok {
				col.Comment = value.GetString()
			}
		}
	}

	return col
}

func (p *mysqlParser) handleAlterTable(cat *catalog.Catalog, n *pcast.AlterTableStmt) {
	schemaName := n.Table.Schema.String()
	if schemaName == "" {
		schemaName = cat.DefaultSchema
	}
	tableName := n.Table.Name.String()

	table := p.findTable(cat, schemaName, tableName)
	if table == nil {
		return
	}

	for _, spec := range n.Specs {
		switch spec.Tp {
		case pcast.AlterTableAddColumns:
			for _, def := range spec.NewColumns {
				col := p.convertColumnDef(def)
				if col != nil {
					exists := false
					for _, existing := range table.Columns {
						if existing.Name == col.Name {
							exists = true
							break
						}
					}
					if !exists {
						table.Columns = append(table.Columns, col)
					}
				}
			}
		case pcast.AlterTableDropColumn:
			name := spec.OldColumnName.String()
			for i, col := range table.Columns {
				if col.Name == name {
					table.Columns = append(table.Columns[:i], table.Columns[i+1:]...)
					break
				}
			}
		case pcast.AlterTableChangeColumn, pcast.AlterTableModifyColumn:
			for _, def := range spec.NewColumns {
				name := def.Name.String()
				for i, col := range table.Columns {
					if col.Name == name || (spec.OldColumnName != nil && col.Name == spec.OldColumnName.String()) {
						newCol := p.convertColumnDef(def)
						if newCol != nil {
							table.Columns[i] = newCol
						}
						break
					}
				}
			}
		case pcast.AlterTableRenameTable:
			if spec.NewTable != nil {
				table.Name = spec.NewTable.Name.String()
			}
		}
	}
}

func (p *mysqlParser) handleDropTable(cat *catalog.Catalog, n *pcast.DropTableStmt) {
	for _, t := range n.Tables {
		schemaName := t.Schema.String()
		if schemaName == "" {
			schemaName = cat.DefaultSchema
		}
		tableName := t.Name.String()

		for _, s := range cat.Schemas {
			if s.Name == schemaName {
				for i, tbl := range s.Tables {
					if strings.EqualFold(tbl.Name, tableName) {
						s.Tables = append(s.Tables[:i], s.Tables[i+1:]...)
						break
					}
				}
			}
		}
	}
}

func (p *mysqlParser) getOrCreateSchema(cat *catalog.Catalog, name string) *catalog.Schema {
	for _, s := range cat.Schemas {
		if s.Name == name {
			return s
		}
	}
	s := &catalog.Schema{Name: name}
	cat.Schemas = append(cat.Schemas, s)
	return s
}

func (p *mysqlParser) findTable(cat *catalog.Catalog, schemaName, tableName string) *catalog.Table {
	for _, s := range cat.Schemas {
		if s.Name == schemaName {
			for _, t := range s.Tables {
				if t.Name == tableName {
					return t
				}
			}
		}
	}
	return nil
}

func (p *mysqlParser) handleCreateView(cat *catalog.Catalog, n *pcast.CreateViewStmt) {
	if n == nil || n.ViewName == nil {
		return
	}

	schemaName := n.ViewName.Schema.String()
	if schemaName == "" {
		schemaName = cat.DefaultSchema
	}
	viewName := n.ViewName.Name.String()

	schema := p.getOrCreateSchema(cat, schemaName)

	// Handle CREATE OR REPLACE
	for i, v := range schema.Views {
		if v.Name == viewName {
			if n.OrReplace {
				schema.Views = append(schema.Views[:i], schema.Views[i+1:]...)
				break
			}
			return
		}
	}

	// Deparse the SELECT query
	query := mysqlRestoreNode(n.Select)

	// Resolve columns
	columns := p.resolveViewColumns(cat, n, schemaName)

	schema.Views = append(schema.Views, &catalog.View{
		Name:    viewName,
		Schema:  schemaName,
		Columns: columns,
		Query:   query,
	})
}

func (p *mysqlParser) resolveViewColumns(cat *catalog.Catalog, n *pcast.CreateViewStmt, schemaName string) []*catalog.Column {
	sel, ok := n.Select.(*pcast.SelectStmt)
	if !ok {
		return nil
	}

	// Collect FROM tables
	fromTables := p.extractFromTables(sel)

	var columns []*catalog.Column
	if sel.Fields == nil {
		return nil
	}

	for i, field := range sel.Fields.Fields {
		// Handle SELECT *
		if field.WildCard != nil {
			tableName := field.WildCard.Table.String()
			expanded := p.expandMysqlStar(cat, tableName, fromTables)
			columns = append(columns, expanded...)
			continue
		}

		colName := field.AsName.String()
		colType := "any"

		// Use explicit view column name if provided
		if i < len(n.Cols) {
			colName = n.Cols[i].String()
		}

		// Try to resolve from ColumnNameExpr
		if cne, ok := field.Expr.(*pcast.ColumnNameExpr); ok && cne.Name != nil {
			if colName == "" {
				colName = cne.Name.Name.String()
			}
			refTable := cne.Name.Table.String()
			if resolved := p.findMysqlColumnType(cat, refTable, cne.Name.Name.String(), fromTables); resolved != "" {
				colType = resolved
			}
		} else if colName == "" {
			colName = fmt.Sprintf("column%d", i+1)
		}

		columns = append(columns, &catalog.Column{
			Name: colName,
			Type: colType,
		})
	}

	return columns
}

func (p *mysqlParser) extractFromTables(sel *pcast.SelectStmt) map[string]string {
	tables := make(map[string]string)
	if sel.From == nil {
		return tables
	}
	p.collectTableSources(sel.From.TableRefs, tables)
	return tables
}

func (p *mysqlParser) collectTableSources(join *pcast.Join, tables map[string]string) {
	if join == nil {
		return
	}
	if join.Left != nil {
		if ts, ok := join.Left.(*pcast.TableSource); ok {
			p.addTableSource(ts, tables)
		} else if j, ok := join.Left.(*pcast.Join); ok {
			p.collectTableSources(j, tables)
		}
	}
	if join.Right != nil {
		if ts, ok := join.Right.(*pcast.TableSource); ok {
			p.addTableSource(ts, tables)
		} else if j, ok := join.Right.(*pcast.Join); ok {
			p.collectTableSources(j, tables)
		}
	}
}

func (p *mysqlParser) addTableSource(ts *pcast.TableSource, tables map[string]string) {
	if ts == nil || ts.Source == nil {
		return
	}
	tn, ok := ts.Source.(*pcast.TableName)
	if !ok {
		return
	}
	name := tn.Name.String()
	alias := name
	if ts.AsName.String() != "" {
		alias = ts.AsName.String()
	}
	tables[alias] = name
}

func (p *mysqlParser) expandMysqlStar(cat *catalog.Catalog, tableName string, fromTables map[string]string) []*catalog.Column {
	var columns []*catalog.Column

	if tableName != "" {
		realName := tableName
		if real, ok := fromTables[tableName]; ok {
			realName = real
		}
		if cols := p.findTableOrViewColumns(cat, realName); cols != nil {
			for _, col := range cols {
				columns = append(columns, &catalog.Column{
					Name:       col.Name,
					Type:       col.Type,
					NotNull:    col.NotNull,
					IsUnsigned: col.IsUnsigned,
				})
			}
		}
		return columns
	}

	for _, realName := range fromTables {
		if cols := p.findTableOrViewColumns(cat, realName); cols != nil {
			for _, col := range cols {
				columns = append(columns, &catalog.Column{
					Name:       col.Name,
					Type:       col.Type,
					NotNull:    col.NotNull,
					IsUnsigned: col.IsUnsigned,
				})
			}
		}
	}
	return columns
}

func (p *mysqlParser) findMysqlColumnType(cat *catalog.Catalog, tableName, colName string, fromTables map[string]string) string {
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

func (p *mysqlParser) findTableOrViewColumns(cat *catalog.Catalog, name string) []*catalog.Column {
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

func (p *mysqlParser) handleDropView(cat *catalog.Catalog, n *pcast.DropTableStmt) {
	for _, t := range n.Tables {
		schemaName := t.Schema.String()
		if schemaName == "" {
			schemaName = cat.DefaultSchema
		}
		viewName := t.Name.String()

		for _, s := range cat.Schemas {
			if s.Name == schemaName {
				for i, v := range s.Views {
					if strings.EqualFold(v.Name, viewName) {
						s.Views = append(s.Views[:i], s.Views[i+1:]...)
						break
					}
				}
			}
		}
	}
}

func mysqlRestoreNode(node pcast.Node) string {
	var sb strings.Builder
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)
	if err := node.Restore(ctx); err != nil {
		return ""
	}
	return sb.String()
}

func mysqlIsNotNull(n *pcast.ColumnDef) bool {
	for _, opt := range n.Options {
		if opt.Tp == pcast.ColumnOptionNotNull {
			return true
		}
		if opt.Tp == pcast.ColumnOptionPrimaryKey {
			return true
		}
	}
	return false
}
