package ddlgen

import (
	"strings"
	"testing"

	"github.com/tkcrm/pgxgen/internal/sqlparser/catalog"
)

func TestPostgresGenerateSimpleTable(t *testing.T) {
	cat := &catalog.Catalog{
		DefaultSchema: "public",
		Schemas: []*catalog.Schema{
			{
				Name: "public",
				Tables: []*catalog.Table{
					{
						Name:   "users",
						Schema: "public",
						PrimaryKey: &catalog.PrimaryKey{
							Columns: []string{"id"},
						},
						Columns: []*catalog.Column{
							{Name: "id", Type: "uuid", FullType: "uuid", NotNull: true, IsPrimary: true, Default: "uuid_generate_v4()"},
							{Name: "name", Type: "varchar", FullType: "character varying(255)", NotNull: true},
							{Name: "email", Type: "text", FullType: "text"},
						},
					},
				},
			},
		},
	}

	gen := &postgresGenerator{}
	ddl, err := gen.Generate(cat)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if !strings.Contains(ddl, `CREATE TABLE "users"`) {
		t.Error("missing CREATE TABLE users")
	}
	if !strings.Contains(ddl, "NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4()") {
		t.Errorf("missing PK + default for id, got:\n%s", ddl)
	}
	if !strings.Contains(ddl, "character varying(255)") {
		t.Error("missing full type for name column")
	}
}

func TestPostgresGenerateWithIndexes(t *testing.T) {
	cat := &catalog.Catalog{
		DefaultSchema: "public",
		Schemas: []*catalog.Schema{
			{
				Name: "public",
				Tables: []*catalog.Table{
					{
						Name:   "users",
						Schema: "public",
						Columns: []*catalog.Column{
							{Name: "id", Type: "serial", FullType: "serial", NotNull: true, IsPrimary: true},
							{Name: "email", Type: "text", FullType: "text", NotNull: true},
						},
						PrimaryKey: &catalog.PrimaryKey{Columns: []string{"id"}},
						Indexes: []*catalog.Index{
							{Name: "idx_email", Columns: []string{"email"}, IsUnique: true, IfNotExists: true, Where: "email <> ''"},
						},
					},
				},
			},
		},
	}

	gen := &postgresGenerator{}
	ddl, err := gen.Generate(cat)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if !strings.Contains(ddl, "CREATE UNIQUE INDEX IF NOT EXISTS") {
		t.Error("missing UNIQUE INDEX IF NOT EXISTS")
	}
	if !strings.Contains(ddl, `WHERE email <> ''`) {
		t.Error("missing WHERE clause")
	}
}

func TestPostgresGenerateWithForeignKeys(t *testing.T) {
	cat := &catalog.Catalog{
		DefaultSchema: "public",
		Schemas: []*catalog.Schema{
			{
				Name: "public",
				Tables: []*catalog.Table{
					{
						Name:       "authors",
						Schema:     "public",
						PrimaryKey: &catalog.PrimaryKey{Columns: []string{"id"}},
						Columns: []*catalog.Column{
							{Name: "id", Type: "uuid", FullType: "uuid", NotNull: true, IsPrimary: true},
						},
					},
					{
						Name:       "books",
						Schema:     "public",
						PrimaryKey: &catalog.PrimaryKey{Columns: []string{"id"}},
						Columns: []*catalog.Column{
							{Name: "id", Type: "uuid", FullType: "uuid", NotNull: true, IsPrimary: true},
							{Name: "author_id", Type: "uuid", FullType: "uuid", NotNull: true},
						},
						ForeignKeys: []*catalog.ForeignKey{
							{Columns: []string{"author_id"}, RefTable: "authors", RefColumns: []string{"id"}},
						},
					},
				},
			},
		},
	}

	gen := &postgresGenerator{}
	ddl, err := gen.Generate(cat)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	// authors should appear before books (FK ordering)
	authorsIdx := strings.Index(ddl, `"authors"`)
	booksIdx := strings.Index(ddl, `"books"`)
	if authorsIdx > booksIdx {
		t.Error("authors should appear before books (FK dependency ordering)")
	}

	if !strings.Contains(ddl, `REFERENCES "authors"`) {
		t.Error("missing FK REFERENCES")
	}
}

func TestPostgresGenerateWithEnums(t *testing.T) {
	cat := &catalog.Catalog{
		DefaultSchema: "public",
		Schemas: []*catalog.Schema{
			{
				Name: "public",
				Enums: []*catalog.Enum{
					{Name: "status", Values: []string{"active", "inactive", "pending"}},
				},
			},
		},
	}

	gen := &postgresGenerator{}
	ddl, err := gen.Generate(cat)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if !strings.Contains(ddl, `CREATE TYPE "status" AS ENUM`) {
		t.Error("missing CREATE TYPE AS ENUM")
	}
	if !strings.Contains(ddl, "'active'") {
		t.Error("missing enum value 'active'")
	}
}

func TestPostgresGenerateWithExtensions(t *testing.T) {
	cat := &catalog.Catalog{
		DefaultSchema: "public",
		Schemas: []*catalog.Schema{
			{
				Name:       "public",
				Extensions: []string{"uuid-ossp", "pgcrypto"},
			},
		},
	}

	gen := &postgresGenerator{}
	ddl, err := gen.Generate(cat)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if !strings.Contains(ddl, `CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`) {
		t.Error("missing uuid-ossp extension")
	}
	if !strings.Contains(ddl, `CREATE EXTENSION IF NOT EXISTS "pgcrypto"`) {
		t.Error("missing pgcrypto extension")
	}
}

func TestPostgresGenerateWithComments(t *testing.T) {
	cat := &catalog.Catalog{
		DefaultSchema: "public",
		Schemas: []*catalog.Schema{
			{
				Name: "public",
				Tables: []*catalog.Table{
					{
						Name:    "users",
						Schema:  "public",
						Comment: "Main users table",
						Columns: []*catalog.Column{
							{Name: "id", Type: "serial", FullType: "serial", NotNull: true},
							{Name: "email", Type: "text", FullType: "text", Comment: "User email"},
						},
					},
				},
			},
		},
	}

	gen := &postgresGenerator{}
	ddl, err := gen.Generate(cat)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if !strings.Contains(ddl, `COMMENT ON TABLE "users" IS 'Main users table'`) {
		t.Error("missing table comment")
	}
	if !strings.Contains(ddl, `COMMENT ON COLUMN "users"."email" IS 'User email'`) {
		t.Error("missing column comment")
	}
}

func TestPostgresGenerateCompositePK(t *testing.T) {
	cat := &catalog.Catalog{
		DefaultSchema: "public",
		Schemas: []*catalog.Schema{
			{
				Name: "public",
				Tables: []*catalog.Table{
					{
						Name:       "order_items",
						Schema:     "public",
						PrimaryKey: &catalog.PrimaryKey{Columns: []string{"order_id", "product_id"}},
						Columns: []*catalog.Column{
							{Name: "order_id", Type: "int4", FullType: "integer", NotNull: true, IsPrimary: true},
							{Name: "product_id", Type: "int4", FullType: "integer", NotNull: true, IsPrimary: true},
							{Name: "quantity", Type: "int4", FullType: "integer", NotNull: true},
						},
					},
				},
			},
		},
	}

	gen := &postgresGenerator{}
	ddl, err := gen.Generate(cat)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	// Should NOT have inline PRIMARY KEY on columns
	if strings.Contains(ddl, `"order_id" integer NOT NULL PRIMARY KEY`) {
		t.Error("composite PK should not be inline")
	}
	// Should have table-level PRIMARY KEY
	if !strings.Contains(ddl, `PRIMARY KEY ("order_id", "product_id")`) {
		t.Errorf("missing composite PK, got:\n%s", ddl)
	}
}

func TestPostgresGenerateWithView(t *testing.T) {
	cat := &catalog.Catalog{
		DefaultSchema: "public",
		Schemas: []*catalog.Schema{
			{
				Name: "public",
				Tables: []*catalog.Table{
					{
						Name:   "users",
						Schema: "public",
						Columns: []*catalog.Column{
							{Name: "id", Type: "serial", FullType: "serial", NotNull: true, IsPrimary: true},
							{Name: "name", Type: "text", FullType: "text", NotNull: true},
						},
					},
				},
				Views: []*catalog.View{
					{
						Name:    "active_users",
						Schema:  "public",
						Comment: "Only active users",
						Query:   "SELECT id, name FROM users WHERE active = true",
						Columns: []*catalog.Column{
							{Name: "id", Type: "serial", NotNull: true},
							{Name: "name", Type: "text", NotNull: true},
						},
					},
				},
			},
		},
	}

	gen := &postgresGenerator{}
	ddl, err := gen.Generate(cat)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if !strings.Contains(ddl, `CREATE VIEW "active_users" AS`) {
		t.Errorf("missing CREATE VIEW, got:\n%s", ddl)
	}
	if !strings.Contains(ddl, "SELECT id, name FROM users WHERE active = true") {
		t.Error("missing view query")
	}
	if !strings.Contains(ddl, `COMMENT ON VIEW "active_users" IS 'Only active users'`) {
		t.Error("missing view comment")
	}
}
