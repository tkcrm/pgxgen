package ddlgen

import (
	"strings"
	"testing"

	"github.com/tkcrm/pgxgen/internal/sqlparser/catalog"
)

func TestSqliteGenerateSimpleTable(t *testing.T) {
	cat := &catalog.Catalog{
		DefaultSchema: "main",
		Schemas: []*catalog.Schema{
			{
				Name: "main",
				Tables: []*catalog.Table{
					{
						Name:       "users",
						Schema:     "main",
						PrimaryKey: &catalog.PrimaryKey{Columns: []string{"id"}},
						Columns: []*catalog.Column{
							{Name: "id", Type: "INTEGER", FullType: "INTEGER", NotNull: true, IsPrimary: true},
							{Name: "name", Type: "TEXT", FullType: "TEXT", NotNull: true},
							{Name: "active", Type: "BOOLEAN", FullType: "BOOLEAN", NotNull: true, Default: "FALSE"},
						},
					},
				},
			},
		},
	}

	gen := &sqliteGenerator{}
	ddl, err := gen.Generate(cat)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if !strings.Contains(ddl, `CREATE TABLE "users"`) {
		t.Error("missing CREATE TABLE")
	}
	if !strings.Contains(ddl, "PRIMARY KEY") {
		t.Error("missing PRIMARY KEY")
	}
	if !strings.Contains(ddl, "DEFAULT FALSE") {
		t.Error("missing DEFAULT FALSE")
	}
}

func TestSqliteGenerateWithIndexes(t *testing.T) {
	cat := &catalog.Catalog{
		DefaultSchema: "main",
		Schemas: []*catalog.Schema{
			{
				Name: "main",
				Tables: []*catalog.Table{
					{
						Name:       "users",
						Schema:     "main",
						PrimaryKey: &catalog.PrimaryKey{Columns: []string{"id"}},
						Columns: []*catalog.Column{
							{Name: "id", Type: "INTEGER", FullType: "INTEGER", NotNull: true, IsPrimary: true},
							{Name: "email", Type: "TEXT", FullType: "TEXT", NotNull: true},
						},
						Indexes: []*catalog.Index{
							{Name: "idx_email", Columns: []string{"email"}, IsUnique: true, IfNotExists: true},
						},
					},
				},
			},
		},
	}

	gen := &sqliteGenerator{}
	ddl, err := gen.Generate(cat)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if !strings.Contains(ddl, "CREATE UNIQUE INDEX IF NOT EXISTS") {
		t.Error("missing UNIQUE INDEX IF NOT EXISTS")
	}
}

func TestSqliteGenerateWithForeignKeys(t *testing.T) {
	cat := &catalog.Catalog{
		DefaultSchema: "main",
		Schemas: []*catalog.Schema{
			{
				Name: "main",
				Tables: []*catalog.Table{
					{
						Name:       "parents",
						Schema:     "main",
						PrimaryKey: &catalog.PrimaryKey{Columns: []string{"id"}},
						Columns: []*catalog.Column{
							{Name: "id", Type: "INTEGER", FullType: "INTEGER", NotNull: true, IsPrimary: true},
						},
					},
					{
						Name:       "children",
						Schema:     "main",
						PrimaryKey: &catalog.PrimaryKey{Columns: []string{"id"}},
						Columns: []*catalog.Column{
							{Name: "id", Type: "INTEGER", FullType: "INTEGER", NotNull: true, IsPrimary: true},
							{Name: "parent_id", Type: "INTEGER", FullType: "INTEGER", NotNull: true},
						},
						ForeignKeys: []*catalog.ForeignKey{
							{Columns: []string{"parent_id"}, RefTable: "parents", RefColumns: []string{"id"}, OnDelete: "CASCADE"},
						},
					},
				},
			},
		},
	}

	gen := &sqliteGenerator{}
	ddl, err := gen.Generate(cat)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	// parents should appear before children
	parentsIdx := strings.Index(ddl, `"parents"`)
	childrenIdx := strings.Index(ddl, `"children"`)
	if parentsIdx > childrenIdx {
		t.Error("parents should appear before children (FK ordering)")
	}

	if !strings.Contains(ddl, `REFERENCES "parents"`) {
		t.Error("missing FK REFERENCES")
	}
	if !strings.Contains(ddl, "ON DELETE CASCADE") {
		t.Error("missing ON DELETE CASCADE")
	}
}

func TestSqliteGenerateCompositePK(t *testing.T) {
	cat := &catalog.Catalog{
		DefaultSchema: "main",
		Schemas: []*catalog.Schema{
			{
				Name: "main",
				Tables: []*catalog.Table{
					{
						Name:       "todo_tags",
						Schema:     "main",
						PrimaryKey: &catalog.PrimaryKey{Columns: []string{"todo_id", "tag_id"}},
						Columns: []*catalog.Column{
							{Name: "todo_id", Type: "TEXT", FullType: "TEXT", NotNull: true, IsPrimary: true},
							{Name: "tag_id", Type: "TEXT", FullType: "TEXT", NotNull: true, IsPrimary: true},
						},
					},
				},
			},
		},
	}

	gen := &sqliteGenerator{}
	ddl, err := gen.Generate(cat)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if strings.Contains(ddl, `"todo_id" TEXT NOT NULL PRIMARY KEY`) {
		t.Error("composite PK should not be inline")
	}
	if !strings.Contains(ddl, `PRIMARY KEY ("todo_id", "tag_id")`) {
		t.Errorf("missing composite PK, got:\n%s", ddl)
	}
}
