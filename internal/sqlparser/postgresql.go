package sqlparser

import (
	"fmt"
	"os"
	"strings"

	pg "github.com/pganalyze/pg_query_go/v6"
	"github.com/tkcrm/pgxgen/internal/sqlparser/catalog"
)

type postgresParser struct{}

func newPostgresParser() *postgresParser {
	return &postgresParser{}
}

func (p *postgresParser) ParseSchema(files []string) (*catalog.Catalog, error) {
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

		result, err := pg.Parse(string(data))
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", file, err)
		}

		for _, stmt := range result.Stmts {
			if stmt.Stmt == nil {
				continue
			}
			switch n := stmt.Stmt.Node.(type) {
			case *pg.Node_CreateStmt:
				p.handleCreateTable(cat, n.CreateStmt)
			case *pg.Node_CreateEnumStmt:
				p.handleCreateEnum(cat, n.CreateEnumStmt)
			case *pg.Node_CommentStmt:
				p.handleComment(cat, n.CommentStmt)
			case *pg.Node_CreateSchemaStmt:
				p.handleCreateSchema(cat, n.CreateSchemaStmt)
			case *pg.Node_AlterTableStmt:
				p.handleAlterTable(cat, n.AlterTableStmt)
			case *pg.Node_DropStmt:
				p.handleDropStmt(cat, n.DropStmt)
			}
		}
	}

	return cat, nil
}

func (p *postgresParser) handleCreateSchema(cat *catalog.Catalog, n *pg.CreateSchemaStmt) {
	if n == nil {
		return
	}
	name := n.Schemaname
	for _, s := range cat.Schemas {
		if s.Name == name {
			return
		}
	}
	cat.Schemas = append(cat.Schemas, &catalog.Schema{Name: name})
}

func (p *postgresParser) handleCreateTable(cat *catalog.Catalog, n *pg.CreateStmt) {
	if n == nil || n.Relation == nil {
		return
	}

	schemaName := n.Relation.Schemaname
	if schemaName == "" {
		schemaName = cat.DefaultSchema
	}
	tableName := n.Relation.Relname

	schema := p.getOrCreateSchema(cat, schemaName)

	// Check if table already exists (IF NOT EXISTS)
	for _, t := range schema.Tables {
		if t.Name == tableName {
			return
		}
	}

	table := &catalog.Table{
		Name:   tableName,
		Schema: schemaName,
	}

	for _, elt := range n.TableElts {
		switch item := elt.Node.(type) {
		case *pg.Node_ColumnDef:
			col := p.convertColumnDef(item.ColumnDef)
			if col != nil {
				table.Columns = append(table.Columns, col)
			}
		}
	}

	schema.Tables = append(schema.Tables, table)
}

func (p *postgresParser) convertColumnDef(n *pg.ColumnDef) *catalog.Column {
	if n == nil {
		return nil
	}

	col := &catalog.Column{
		Name:    n.Colname,
		NotNull: n.IsNotNull,
	}

	if n.TypeName != nil {
		col.Type = pgTypeName(n.TypeName)
		col.IsArray = len(n.TypeName.ArrayBounds) > 0
		col.ArrayDims = len(n.TypeName.ArrayBounds)
	}

	// Check constraints for NOT NULL / PRIMARY KEY
	for _, c := range n.Constraints {
		if constraint, ok := c.Node.(*pg.Node_Constraint); ok {
			switch constraint.Constraint.Contype {
			case pg.ConstrType_CONSTR_NOTNULL:
				col.NotNull = true
			case pg.ConstrType_CONSTR_PRIMARY:
				col.NotNull = true
			}
		}
	}

	return col
}

func (p *postgresParser) handleCreateEnum(cat *catalog.Catalog, n *pg.CreateEnumStmt) {
	if n == nil {
		return
	}

	schemaName, enumName := pgParseTypeName(n.TypeName)
	if schemaName == "" {
		schemaName = cat.DefaultSchema
	}

	schema := p.getOrCreateSchema(cat, schemaName)

	enum := &catalog.Enum{
		Name: enumName,
	}
	for _, val := range n.Vals {
		if s, ok := val.Node.(*pg.Node_String_); ok {
			enum.Values = append(enum.Values, s.String_.Sval)
		}
	}

	schema.Enums = append(schema.Enums, enum)
}

func (p *postgresParser) handleComment(cat *catalog.Catalog, n *pg.CommentStmt) {
	if n == nil || n.Object == nil {
		return
	}

	comment := n.Comment

	switch n.Objtype {
	case pg.ObjectType_OBJECT_TABLE:
		// Object is a list of names [schema, table]
		if list, ok := n.Object.Node.(*pg.Node_List); ok {
			schemaName, tableName := pgParseQualifiedName(list.List)
			if schemaName == "" {
				schemaName = cat.DefaultSchema
			}
			for _, s := range cat.Schemas {
				if s.Name == schemaName {
					for _, t := range s.Tables {
						if t.Name == tableName {
							t.Comment = comment
							return
						}
					}
				}
			}
		}

	case pg.ObjectType_OBJECT_COLUMN:
		// Object is a list of names [schema, table, column]
		if list, ok := n.Object.Node.(*pg.Node_List); ok {
			items := list.List.Items
			var schemaName, tableName, colName string
			switch len(items) {
			case 3:
				schemaName = pgStringVal(items[0])
				tableName = pgStringVal(items[1])
				colName = pgStringVal(items[2])
			case 2:
				schemaName = cat.DefaultSchema
				tableName = pgStringVal(items[0])
				colName = pgStringVal(items[1])
			}
			for _, s := range cat.Schemas {
				if s.Name == schemaName {
					for _, t := range s.Tables {
						if t.Name == tableName {
							for _, c := range t.Columns {
								if c.Name == colName {
									c.Comment = comment
									return
								}
							}
						}
					}
				}
			}
		}
	}
}

func (p *postgresParser) handleAlterTable(cat *catalog.Catalog, n *pg.AlterTableStmt) {
	if n == nil || n.Relation == nil {
		return
	}

	schemaName := n.Relation.Schemaname
	if schemaName == "" {
		schemaName = cat.DefaultSchema
	}
	tableName := n.Relation.Relname

	table := p.findTable(cat, schemaName, tableName)
	if table == nil {
		return
	}

	for _, cmd := range n.Cmds {
		altCmd, ok := cmd.Node.(*pg.Node_AlterTableCmd)
		if !ok {
			continue
		}
		c := altCmd.AlterTableCmd
		switch c.Subtype {
		case pg.AlterTableType_AT_AddColumn:
			if def, ok := c.Def.Node.(*pg.Node_ColumnDef); ok {
				col := p.convertColumnDef(def.ColumnDef)
				if col != nil {
					table.Columns = append(table.Columns, col)
				}
			}
		case pg.AlterTableType_AT_DropColumn:
			name := c.Name
			for i, col := range table.Columns {
				if col.Name == name {
					table.Columns = append(table.Columns[:i], table.Columns[i+1:]...)
					break
				}
			}
		case pg.AlterTableType_AT_SetNotNull:
			for _, col := range table.Columns {
				if col.Name == c.Name {
					col.NotNull = true
					break
				}
			}
		case pg.AlterTableType_AT_DropNotNull:
			for _, col := range table.Columns {
				if col.Name == c.Name {
					col.NotNull = false
					break
				}
			}
		case pg.AlterTableType_AT_AlterColumnType:
			if def, ok := c.Def.Node.(*pg.Node_ColumnDef); ok {
				for _, col := range table.Columns {
					if col.Name == def.ColumnDef.Colname {
						if def.ColumnDef.TypeName != nil {
							col.Type = pgTypeName(def.ColumnDef.TypeName)
							col.IsArray = len(def.ColumnDef.TypeName.ArrayBounds) > 0
							col.ArrayDims = len(def.ColumnDef.TypeName.ArrayBounds)
						}
						break
					}
				}
			}
		}
	}
}

func (p *postgresParser) handleDropStmt(cat *catalog.Catalog, n *pg.DropStmt) {
	if n == nil {
		return
	}

	switch n.RemoveType {
	case pg.ObjectType_OBJECT_TABLE:
		for _, obj := range n.Objects {
			if list, ok := obj.Node.(*pg.Node_List); ok {
				schemaName, tableName := pgParseQualifiedName(list.List)
				if schemaName == "" {
					schemaName = cat.DefaultSchema
				}
				for _, s := range cat.Schemas {
					if s.Name == schemaName {
						for i, t := range s.Tables {
							if t.Name == tableName {
								s.Tables = append(s.Tables[:i], s.Tables[i+1:]...)
								break
							}
						}
					}
				}
			}
		}
	case pg.ObjectType_OBJECT_TYPE:
		for _, obj := range n.Objects {
			if tn, ok := obj.Node.(*pg.Node_TypeName); ok {
				schemaName, enumName := pgParseTypeName(tn.TypeName.Names)
				if schemaName == "" {
					schemaName = cat.DefaultSchema
				}
				for _, s := range cat.Schemas {
					if s.Name == schemaName {
						for i, e := range s.Enums {
							if e.Name == enumName {
								s.Enums = append(s.Enums[:i], s.Enums[i+1:]...)
								break
							}
						}
					}
				}
			}
		}
	}
}

func (p *postgresParser) getOrCreateSchema(cat *catalog.Catalog, name string) *catalog.Schema {
	for _, s := range cat.Schemas {
		if s.Name == name {
			return s
		}
	}
	s := &catalog.Schema{Name: name}
	cat.Schemas = append(cat.Schemas, s)
	return s
}

func (p *postgresParser) findTable(cat *catalog.Catalog, schemaName, tableName string) *catalog.Table {
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

// pgTypeName extracts the type name string from a pg_query TypeName node.
func pgTypeName(tn *pg.TypeName) string {
	if tn == nil {
		return ""
	}

	var parts []string
	for _, n := range tn.Names {
		if s, ok := n.Node.(*pg.Node_String_); ok {
			val := s.String_.Sval
			// Skip the "pg_catalog" schema prefix for built-in types
			if val == "pg_catalog" {
				continue
			}
			parts = append(parts, val)
		}
	}

	name := strings.Join(parts, ".")
	if name == "" {
		return ""
	}

	return name
}

// pgParseTypeName parses a list of name nodes into schema and type name.
func pgParseTypeName(names []*pg.Node) (schema, name string) {
	var parts []string
	for _, n := range names {
		if s, ok := n.Node.(*pg.Node_String_); ok {
			parts = append(parts, s.String_.Sval)
		}
	}
	switch len(parts) {
	case 1:
		return "", parts[0]
	case 2:
		return parts[0], parts[1]
	default:
		return "", strings.Join(parts, ".")
	}
}

// pgParseQualifiedName parses a List node into schema and object name.
func pgParseQualifiedName(list *pg.List) (schema, name string) {
	items := list.Items
	switch len(items) {
	case 1:
		return "", pgStringVal(items[0])
	case 2:
		return pgStringVal(items[0]), pgStringVal(items[1])
	default:
		return "", ""
	}
}

func pgStringVal(n *pg.Node) string {
	if s, ok := n.Node.(*pg.Node_String_); ok {
		return s.String_.Sval
	}
	return ""
}
