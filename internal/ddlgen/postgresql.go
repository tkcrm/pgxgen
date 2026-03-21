package ddlgen

import (
	"fmt"
	"strings"

	"github.com/tkcrm/pgxgen/internal/sqlparser/catalog"
)

type postgresGenerator struct{}

func (g *postgresGenerator) Generate(cat *catalog.Catalog) (string, error) {
	var buf strings.Builder

	for _, schema := range cat.Schemas {
		g.writeExtensions(&buf, schema)
		g.writeNonDefaultSchema(&buf, schema, cat.DefaultSchema)
		g.writeEnums(&buf, schema)

		// Sort tables by FK dependencies (topological order)
		tables := topologicalSort(schema.Tables)
		for _, table := range tables {
			g.writeTable(&buf, table)
			g.writeIndexes(&buf, table)
			g.writeComments(&buf, table)
		}

		for _, view := range schema.Views {
			g.writeView(&buf, view)
		}
	}

	return strings.TrimRight(buf.String(), "\n") + "\n", nil
}

func (g *postgresGenerator) writeExtensions(buf *strings.Builder, schema *catalog.Schema) {
	for _, ext := range schema.Extensions {
		fmt.Fprintf(buf, "CREATE EXTENSION IF NOT EXISTS %s;\n\n", pgQuoteIdent(ext))
	}
}

func (g *postgresGenerator) writeNonDefaultSchema(buf *strings.Builder, schema *catalog.Schema, defaultSchema string) {
	if schema.Name != defaultSchema {
		fmt.Fprintf(buf, "CREATE SCHEMA IF NOT EXISTS %s;\n\n", pgQuoteIdent(schema.Name))
	}
}

func (g *postgresGenerator) writeEnums(buf *strings.Builder, schema *catalog.Schema) {
	for _, enum := range schema.Enums {
		fmt.Fprintf(buf, "CREATE TYPE %s AS ENUM (\n", pgQuoteIdent(enum.Name))
		for i, val := range enum.Values {
			fmt.Fprintf(buf, "    '%s'", strings.ReplaceAll(val, "'", "''"))
			if i < len(enum.Values)-1 {
				buf.WriteString(",")
			}
			buf.WriteString("\n")
		}
		buf.WriteString(");\n\n")
	}
}

func (g *postgresGenerator) writeTable(buf *strings.Builder, table *catalog.Table) {
	fmt.Fprintf(buf, "CREATE TABLE %s (\n", pgQuoteIdent(table.Name))

	// Collect all lines (columns + constraints)
	var lines []string

	for _, col := range table.Columns {
		lines = append(lines, g.columnLine(col, table))
	}

	// Table-level PRIMARY KEY (only for composite PKs)
	if table.PrimaryKey != nil && len(table.PrimaryKey.Columns) > 1 {
		line := fmt.Sprintf("    PRIMARY KEY (%s)", joinQuoted(table.PrimaryKey.Columns))
		if table.PrimaryKey.Name != "" {
			line = fmt.Sprintf("    CONSTRAINT %s PRIMARY KEY (%s)", pgQuoteIdent(table.PrimaryKey.Name), joinQuoted(table.PrimaryKey.Columns))
		}
		lines = append(lines, line)
	}

	// UNIQUE constraints
	for _, u := range table.Uniques {
		line := fmt.Sprintf("    UNIQUE (%s)", joinQuoted(u.Columns))
		if u.Name != "" {
			line = fmt.Sprintf("    CONSTRAINT %s UNIQUE (%s)", pgQuoteIdent(u.Name), joinQuoted(u.Columns))
		}
		lines = append(lines, line)
	}

	// FOREIGN KEY constraints (table-level, for multi-column FKs or named FKs without inline)
	for _, fk := range table.ForeignKeys {
		// Only write as table-level if multi-column or has explicit constraint name
		if len(fk.Columns) > 1 || fk.Name != "" {
			lines = append(lines, g.fkConstraintLine(fk))
		}
	}

	// CHECK constraints
	for _, c := range table.Checks {
		line := fmt.Sprintf("    CHECK (%s)", c.Expression)
		if c.Name != "" {
			line = fmt.Sprintf("    CONSTRAINT %s CHECK (%s)", pgQuoteIdent(c.Name), c.Expression)
		}
		lines = append(lines, line)
	}

	for i, line := range lines {
		buf.WriteString(line)
		if i < len(lines)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}

	buf.WriteString(");\n\n")
}

func (g *postgresGenerator) columnLine(col *catalog.Column, table *catalog.Table) string {
	var parts []string
	parts = append(parts, "    "+pgQuoteIdent(col.Name))

	// Type with array dimensions — prefer FullType (includes length/precision)
	typeName := col.FullType
	if typeName == "" {
		typeName = col.Type
	}
	if col.IsArray {
		typeName += strings.Repeat("[]", col.ArrayDims)
	}
	parts = append(parts, typeName)

	if col.NotNull {
		parts = append(parts, "NOT NULL")
	}

	// Inline PRIMARY KEY for single-column PKs
	if col.IsPrimary && table.PrimaryKey != nil && len(table.PrimaryKey.Columns) == 1 {
		parts = append(parts, "PRIMARY KEY")
	}

	if col.Default != "" {
		parts = append(parts, "DEFAULT "+col.Default)
	}

	// Inline REFERENCES for single-column FK without explicit name
	for _, fk := range table.ForeignKeys {
		if len(fk.Columns) == 1 && fk.Columns[0] == col.Name && fk.Name == "" {
			ref := fmt.Sprintf("REFERENCES %s", pgQuoteIdent(fk.RefTable))
			if len(fk.RefColumns) > 0 {
				ref += fmt.Sprintf(" (%s)", joinQuoted(fk.RefColumns))
			}
			if fk.OnDelete != "" && fk.OnDelete != "NO ACTION" {
				ref += " ON DELETE " + fk.OnDelete
			}
			if fk.OnUpdate != "" && fk.OnUpdate != "NO ACTION" {
				ref += " ON UPDATE " + fk.OnUpdate
			}
			parts = append(parts, ref)
		}
	}

	return strings.Join(parts, " ")
}

func (g *postgresGenerator) fkConstraintLine(fk *catalog.ForeignKey) string {
	var sb strings.Builder
	sb.WriteString("    ")
	if fk.Name != "" {
		fmt.Fprintf(&sb, "CONSTRAINT %s ", pgQuoteIdent(fk.Name))
	}
	fmt.Fprintf(&sb, "FOREIGN KEY (%s) REFERENCES %s", joinQuoted(fk.Columns), pgQuoteIdent(fk.RefTable))
	if len(fk.RefColumns) > 0 {
		fmt.Fprintf(&sb, " (%s)", joinQuoted(fk.RefColumns))
	}
	if fk.OnDelete != "" && fk.OnDelete != "NO ACTION" {
		sb.WriteString(" ON DELETE " + fk.OnDelete)
	}
	if fk.OnUpdate != "" && fk.OnUpdate != "NO ACTION" {
		sb.WriteString(" ON UPDATE " + fk.OnUpdate)
	}
	return sb.String()
}

func (g *postgresGenerator) writeIndexes(buf *strings.Builder, table *catalog.Table) {
	for _, idx := range table.Indexes {
		if idx.IsUnique {
			buf.WriteString("CREATE UNIQUE INDEX ")
		} else {
			buf.WriteString("CREATE INDEX ")
		}
		if idx.IfNotExists {
			buf.WriteString("IF NOT EXISTS ")
		}
		fmt.Fprintf(buf, "%s ON %s (%s)", pgQuoteIdent(idx.Name), pgQuoteIdent(table.Name), joinQuoted(idx.Columns))
		if idx.Where != "" {
			fmt.Fprintf(buf, " WHERE %s", idx.Where)
		}
		buf.WriteString(";\n")
	}
	if len(table.Indexes) > 0 {
		buf.WriteString("\n")
	}
}

func (g *postgresGenerator) writeComments(buf *strings.Builder, table *catalog.Table) {
	hasComments := false
	if table.Comment != "" {
		fmt.Fprintf(buf, "COMMENT ON TABLE %s IS '%s';\n", pgQuoteIdent(table.Name), strings.ReplaceAll(table.Comment, "'", "''"))
		hasComments = true
	}
	for _, col := range table.Columns {
		if col.Comment != "" {
			fmt.Fprintf(buf, "COMMENT ON COLUMN %s.%s IS '%s';\n", pgQuoteIdent(table.Name), pgQuoteIdent(col.Name), strings.ReplaceAll(col.Comment, "'", "''"))
			hasComments = true
		}
	}
	if hasComments {
		buf.WriteString("\n")
	}
}

func (g *postgresGenerator) writeView(buf *strings.Builder, view *catalog.View) {
	fmt.Fprintf(buf, "CREATE VIEW %s AS\n%s;\n\n", pgQuoteIdent(view.Name), view.Query)
	if view.Comment != "" {
		fmt.Fprintf(buf, "COMMENT ON VIEW %s IS '%s';\n\n",
			pgQuoteIdent(view.Name), strings.ReplaceAll(view.Comment, "'", "''"))
	}
}

// pgQuoteIdent quotes a PostgreSQL identifier with double quotes.
func pgQuoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// joinQuoted joins identifiers with commas, each quoted.
func joinQuoted(names []string) string {
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = pgQuoteIdent(n)
	}
	return strings.Join(quoted, ", ")
}

// topologicalSort orders tables so that tables referenced by foreign keys
// come before the tables that reference them.
func topologicalSort(tables []*catalog.Table) []*catalog.Table {
	if len(tables) <= 1 {
		return tables
	}

	// Build adjacency: table -> tables it depends on (via FK)
	tableMap := make(map[string]*catalog.Table)
	deps := make(map[string]map[string]bool)
	for _, t := range tables {
		tableMap[t.Name] = t
		deps[t.Name] = make(map[string]bool)
		for _, fk := range t.ForeignKeys {
			if fk.RefTable != t.Name { // skip self-references
				deps[t.Name][fk.RefTable] = true
			}
		}
	}

	var result []*catalog.Table
	visited := make(map[string]bool)
	visiting := make(map[string]bool) // cycle detection

	var visit func(name string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		if visiting[name] {
			// Cycle detected — break it
			visited[name] = true
			return
		}
		visiting[name] = true
		for dep := range deps[name] {
			if tableMap[dep] != nil {
				visit(dep)
			}
		}
		visiting[name] = false
		visited[name] = true
		if t, ok := tableMap[name]; ok {
			result = append(result, t)
		}
	}

	for _, t := range tables {
		visit(t.Name)
	}

	return result
}
