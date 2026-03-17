package sqlparser

import (
	"fmt"
	"os"
	"strings"

	pcast "github.com/pingcap/tidb/pkg/parser/ast"
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
				p.handleDropTable(cat, n)
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
					table.Columns = append(table.Columns, col)
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
