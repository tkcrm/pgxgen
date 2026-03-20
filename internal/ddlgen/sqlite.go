package ddlgen

import (
	"fmt"
	"strings"

	"github.com/tkcrm/pgxgen/internal/sqlparser/catalog"
)

type sqliteGenerator struct{}

func (g *sqliteGenerator) Generate(cat *catalog.Catalog) (string, error) {
	var buf strings.Builder

	for _, schema := range cat.Schemas {
		tables := topologicalSort(schema.Tables)
		for _, table := range tables {
			g.writeTable(&buf, table)
			g.writeIndexes(&buf, table)
		}
	}

	return strings.TrimRight(buf.String(), "\n") + "\n", nil
}

func (g *sqliteGenerator) writeTable(buf *strings.Builder, table *catalog.Table) {
	fmt.Fprintf(buf, "CREATE TABLE %s (\n", pgQuoteIdent(table.Name))

	var lines []string

	for _, col := range table.Columns {
		lines = append(lines, g.columnLine(col, table))
	}

	// Table-level PRIMARY KEY (composite)
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

	// FOREIGN KEY constraints (table-level: multi-column or named)
	for _, fk := range table.ForeignKeys {
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

func (g *sqliteGenerator) columnLine(col *catalog.Column, table *catalog.Table) string {
	var parts []string
	parts = append(parts, "    "+pgQuoteIdent(col.Name))

	typeName := col.FullType
	if typeName == "" {
		typeName = col.Type
	}
	parts = append(parts, typeName)

	if col.NotNull {
		parts = append(parts, "NOT NULL")
	}

	// Inline PRIMARY KEY for single-column PK
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
				ref += fmt.Sprintf("(%s)", joinQuoted(fk.RefColumns))
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

func (g *sqliteGenerator) fkConstraintLine(fk *catalog.ForeignKey) string {
	var sb strings.Builder
	sb.WriteString("    ")
	if fk.Name != "" {
		fmt.Fprintf(&sb, "CONSTRAINT %s ", pgQuoteIdent(fk.Name))
	}
	fmt.Fprintf(&sb, "FOREIGN KEY (%s) REFERENCES %s", joinQuoted(fk.Columns), pgQuoteIdent(fk.RefTable))
	if len(fk.RefColumns) > 0 {
		fmt.Fprintf(&sb, "(%s)", joinQuoted(fk.RefColumns))
	}
	if fk.OnDelete != "" && fk.OnDelete != "NO ACTION" {
		sb.WriteString(" ON DELETE " + fk.OnDelete)
	}
	if fk.OnUpdate != "" && fk.OnUpdate != "NO ACTION" {
		sb.WriteString(" ON UPDATE " + fk.OnUpdate)
	}
	return sb.String()
}

func (g *sqliteGenerator) writeIndexes(buf *strings.Builder, table *catalog.Table) {
	for _, idx := range table.Indexes {
		if idx.IsUnique {
			buf.WriteString("CREATE UNIQUE INDEX ")
		} else {
			buf.WriteString("CREATE INDEX ")
		}
		if idx.IfNotExists {
			buf.WriteString("IF NOT EXISTS ")
		}
		fmt.Fprintf(buf, "%s ON %s(%s)", pgQuoteIdent(idx.Name), pgQuoteIdent(table.Name), joinQuoted(idx.Columns))
		if idx.Where != "" {
			fmt.Fprintf(buf, " WHERE %s", idx.Where)
		}
		buf.WriteString(";\n")
	}
	if len(table.Indexes) > 0 {
		buf.WriteString("\n")
	}
}
