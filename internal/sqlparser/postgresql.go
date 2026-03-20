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
			case *pg.Node_IndexStmt:
				p.handleCreateIndex(cat, n.IndexStmt)
			case *pg.Node_CreateExtensionStmt:
				p.handleCreateExtension(cat, n.CreateExtensionStmt)
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

	// Collect inline foreign keys from column REFERENCES constraints.
	// We need to process these after all columns are added.
	type inlineFK struct {
		colName    string
		constraint *pg.Constraint
	}
	var inlineFKs []inlineFK

	for _, elt := range n.TableElts {
		switch item := elt.Node.(type) {
		case *pg.Node_ColumnDef:
			col := p.convertColumnDef(item.ColumnDef)
			if col != nil {
				table.Columns = append(table.Columns, col)
			}
			// Check for inline REFERENCES constraint
			for _, c := range item.ColumnDef.Constraints {
				if constraint, ok := c.Node.(*pg.Node_Constraint); ok {
					if constraint.Constraint.Contype == pg.ConstrType_CONSTR_FOREIGN {
						inlineFKs = append(inlineFKs, inlineFK{
							colName:    item.ColumnDef.Colname,
							constraint: constraint.Constraint,
						})
					}
				}
			}
		case *pg.Node_Constraint:
			p.handleTableConstraint(table, item.Constraint)
		}
	}

	// Process inline foreign keys
	for _, ifk := range inlineFKs {
		fk := p.convertForeignKey(ifk.constraint)
		if fk != nil {
			fk.Columns = []string{ifk.colName}
			table.ForeignKeys = append(table.ForeignKeys, fk)
		}
	}

	// Build PrimaryKey from inline column-level PRIMARY KEY if no table-level PK was set
	if table.PrimaryKey == nil {
		var pkCols []string
		for _, col := range table.Columns {
			if col.IsPrimary {
				pkCols = append(pkCols, col.Name)
			}
		}
		if len(pkCols) > 0 {
			table.PrimaryKey = &catalog.PrimaryKey{Columns: pkCols}
		}
	}

	schema.Tables = append(schema.Tables, table)
}

// handleTableConstraint processes a table-level constraint node.
func (p *postgresParser) handleTableConstraint(table *catalog.Table, n *pg.Constraint) {
	if n == nil {
		return
	}
	switch n.Contype {
	case pg.ConstrType_CONSTR_PRIMARY:
		pk := &catalog.PrimaryKey{Name: n.Conname}
		for _, key := range n.Keys {
			pk.Columns = append(pk.Columns, pgStringVal(key))
		}
		table.PrimaryKey = pk
		// Mark columns as NOT NULL
		for _, pkCol := range pk.Columns {
			for _, col := range table.Columns {
				if col.Name == pkCol {
					col.NotNull = true
					col.IsPrimary = true
				}
			}
		}
	case pg.ConstrType_CONSTR_UNIQUE:
		uc := &catalog.UniqueConstraint{Name: n.Conname}
		for _, key := range n.Keys {
			uc.Columns = append(uc.Columns, pgStringVal(key))
		}
		table.Uniques = append(table.Uniques, uc)
	case pg.ConstrType_CONSTR_FOREIGN:
		fk := p.convertForeignKey(n)
		if fk != nil {
			// Table-level FK: columns from FkAttrs
			for _, attr := range n.FkAttrs {
				fk.Columns = append(fk.Columns, pgStringVal(attr))
			}
			table.ForeignKeys = append(table.ForeignKeys, fk)
		}
	case pg.ConstrType_CONSTR_CHECK:
		cc := &catalog.CheckConstraint{Name: n.Conname}
		if n.RawExpr != nil {
			cc.Expression = deparseExpr(n.RawExpr)
		}
		table.Checks = append(table.Checks, cc)
	}
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
		col.FullType = deparseTypeName(n.TypeName)
		col.IsArray = len(n.TypeName.ArrayBounds) > 0
		col.ArrayDims = len(n.TypeName.ArrayBounds)
	}

	// Check constraints for NOT NULL / PRIMARY KEY / DEFAULT
	for _, c := range n.Constraints {
		if constraint, ok := c.Node.(*pg.Node_Constraint); ok {
			switch constraint.Constraint.Contype {
			case pg.ConstrType_CONSTR_NOTNULL:
				col.NotNull = true
			case pg.ConstrType_CONSTR_PRIMARY:
				col.NotNull = true
				col.IsPrimary = true
			case pg.ConstrType_CONSTR_DEFAULT:
				if constraint.Constraint.RawExpr != nil {
					col.Default = deparseExpr(constraint.Constraint.RawExpr)
				}
			}
		}
	}

	return col
}

// convertForeignKey extracts FK details from a Constraint node.
func (p *postgresParser) convertForeignKey(n *pg.Constraint) *catalog.ForeignKey {
	if n == nil || n.Pktable == nil {
		return nil
	}
	fk := &catalog.ForeignKey{
		Name:     n.Conname,
		RefTable: n.Pktable.Relname,
	}
	if n.Pktable.Schemaname != "" {
		fk.RefSchema = n.Pktable.Schemaname
	}
	for _, attr := range n.PkAttrs {
		fk.RefColumns = append(fk.RefColumns, pgStringVal(attr))
	}
	fk.OnDelete = pgFKAction(n.FkDelAction)
	fk.OnUpdate = pgFKAction(n.FkUpdAction)
	return fk
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
							col.FullType = deparseTypeName(def.ColumnDef.TypeName)
							col.IsArray = len(def.ColumnDef.TypeName.ArrayBounds) > 0
							col.ArrayDims = len(def.ColumnDef.TypeName.ArrayBounds)
						}
						break
					}
				}
			}
		case pg.AlterTableType_AT_ColumnDefault:
			for _, col := range table.Columns {
				if col.Name == c.Name {
					if c.Def != nil {
						col.Default = deparseExpr(c.Def)
					} else {
						col.Default = ""
					}
					break
				}
			}
		case pg.AlterTableType_AT_AddConstraint:
			if constraint, ok := c.Def.Node.(*pg.Node_Constraint); ok {
				p.handleTableConstraint(table, constraint.Constraint)
			}
		case pg.AlterTableType_AT_DropConstraint:
			p.dropConstraint(table, c.Name)
		}
	}
}

func (p *postgresParser) handleCreateIndex(cat *catalog.Catalog, n *pg.IndexStmt) {
	if n == nil || n.Relation == nil {
		return
	}

	schemaName := n.Relation.Schemaname
	if schemaName == "" {
		schemaName = cat.DefaultSchema
	}

	table := p.findTable(cat, schemaName, n.Relation.Relname)
	if table == nil {
		return
	}

	idx := &catalog.Index{
		Name:        n.Idxname,
		IsUnique:    n.Unique,
		IfNotExists: n.IfNotExists,
	}

	// Extract column names from index params
	for _, param := range n.IndexParams {
		if elem, ok := param.Node.(*pg.Node_IndexElem); ok {
			if elem.IndexElem.Name != "" {
				idx.Columns = append(idx.Columns, elem.IndexElem.Name)
			} else if elem.IndexElem.Expr != nil {
				// Expression index — deparse the expression
				idx.Columns = append(idx.Columns, deparseExpr(elem.IndexElem.Expr))
			}
		}
	}

	// Partial index WHERE clause
	if n.WhereClause != nil {
		idx.Where = deparseExpr(n.WhereClause)
	}

	table.Indexes = append(table.Indexes, idx)
}

func (p *postgresParser) handleCreateExtension(cat *catalog.Catalog, n *pg.CreateExtensionStmt) {
	if n == nil {
		return
	}

	schema := p.getOrCreateSchema(cat, cat.DefaultSchema)

	// Avoid duplicates
	for _, ext := range schema.Extensions {
		if ext == n.Extname {
			return
		}
	}
	schema.Extensions = append(schema.Extensions, n.Extname)
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
	case pg.ObjectType_OBJECT_INDEX:
		for _, obj := range n.Objects {
			if list, ok := obj.Node.(*pg.Node_List); ok {
				_, idxName := pgParseQualifiedName(list.List)
				// Search all tables for this index
				for _, s := range cat.Schemas {
					for _, t := range s.Tables {
						for i, idx := range t.Indexes {
							if idx.Name == idxName {
								t.Indexes = append(t.Indexes[:i], t.Indexes[i+1:]...)
								break
							}
						}
					}
				}
			}
		}
	}
}

// dropConstraint removes a constraint by name from a table.
func (p *postgresParser) dropConstraint(table *catalog.Table, name string) {
	if name == "" {
		return
	}
	if table.PrimaryKey != nil && table.PrimaryKey.Name == name {
		table.PrimaryKey = nil
		return
	}
	for i, u := range table.Uniques {
		if u.Name == name {
			table.Uniques = append(table.Uniques[:i], table.Uniques[i+1:]...)
			return
		}
	}
	for i, fk := range table.ForeignKeys {
		if fk.Name == name {
			table.ForeignKeys = append(table.ForeignKeys[:i], table.ForeignKeys[i+1:]...)
			return
		}
	}
	for i, c := range table.Checks {
		if c.Name == name {
			table.Checks = append(table.Checks[:i], table.Checks[i+1:]...)
			return
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

// deparseTypeName converts a pg_query TypeName node to its full SQL representation
// including modifiers like length, e.g. "character varying(255)".
func deparseTypeName(tn *pg.TypeName) string {
	if tn == nil {
		return ""
	}
	// Build a synthetic "SELECT NULL::<type>" and deparse, then extract the type part.
	colDef := &pg.ColumnDef{
		Colname:  "_",
		TypeName: tn,
	}
	createStmt := &pg.CreateStmt{
		Relation: &pg.RangeVar{Relname: "_"},
		TableElts: []*pg.Node{
			{Node: &pg.Node_ColumnDef{ColumnDef: colDef}},
		},
	}
	tree := &pg.ParseResult{
		Stmts: []*pg.RawStmt{
			{Stmt: &pg.Node{Node: &pg.Node_CreateStmt{CreateStmt: createStmt}}},
		},
	}
	output, err := pg.Deparse(tree)
	if err != nil {
		return pgTypeName(tn)
	}
	// output: CREATE TABLE _ (_ <type>)
	// Extract the type between "(_ " and ")"
	start := strings.Index(output, "(_ ")
	end := strings.LastIndex(output, ")")
	if start >= 0 && end > start+3 {
		return strings.TrimSpace(output[start+3 : end])
	}
	return pgTypeName(tn)
}

// deparseExpr converts a pg_query AST node back to SQL text.
// It wraps the expression in a synthetic SELECT statement, deparses it,
// then strips the "SELECT " prefix.
func deparseExpr(node *pg.Node) string {
	if node == nil {
		return ""
	}
	selectStmt := &pg.SelectStmt{
		TargetList: []*pg.Node{
			{Node: &pg.Node_ResTarget{ResTarget: &pg.ResTarget{Val: node}}},
		},
	}
	tree := &pg.ParseResult{
		Stmts: []*pg.RawStmt{
			{Stmt: &pg.Node{Node: &pg.Node_SelectStmt{SelectStmt: selectStmt}}},
		},
	}
	output, err := pg.Deparse(tree)
	if err != nil {
		return ""
	}
	// Strip "SELECT " prefix
	return strings.TrimPrefix(output, "SELECT ")
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

// pgFKAction converts a pg_query FK action character to a human-readable string.
func pgFKAction(action string) string {
	switch action {
	case "a":
		return "NO ACTION"
	case "r":
		return "RESTRICT"
	case "c":
		return "CASCADE"
	case "n":
		return "SET NULL"
	case "d":
		return "SET DEFAULT"
	default:
		return ""
	}
}
