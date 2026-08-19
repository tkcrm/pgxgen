package sqlparser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tkcrm/pgxgen/internal/sqlparser/catalog"
	"github.com/tkcrm/pgxgen/internal/sqlparser/typemap"
)

func writeTemp(t *testing.T, sql string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "schema.sql")
	if err := os.WriteFile(path, []byte(sql), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPostgresCreateTable(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE users (
			id UUID NOT NULL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email TEXT NULL,
			age INTEGER NOT NULL DEFAULT 0,
			score BIGINT NULL,
			rating REAL NOT NULL,
			balance NUMERIC(10,2) NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT FALSE,
			data JSONB NULL,
			avatar BYTEA NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMPTZ NULL,
			small_id SMALLINT NOT NULL,
			auto_id SERIAL NOT NULL,
			big_auto BIGSERIAL NOT NULL
		);
	`)

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	if len(cat.Schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(cat.Schemas))
	}
	if cat.Schemas[0].Name != "public" {
		t.Errorf("expected schema 'public', got %q", cat.Schemas[0].Name)
	}

	tables := cat.Schemas[0].Tables
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}

	table := tables[0]
	if table.Name != "users" {
		t.Errorf("expected table 'users', got %q", table.Name)
	}
	if table.Schema != "public" {
		t.Errorf("expected schema 'public', got %q", table.Schema)
	}

	if len(table.Columns) != 15 {
		t.Fatalf("expected 15 columns, got %d", len(table.Columns))
	}

	// Verify specific columns
	tests := []struct {
		idx     int
		name    string
		typ     string
		notNull bool
	}{
		{0, "id", "uuid", true},
		{1, "name", "varchar", true},
		{2, "email", "text", false},
		{3, "age", "int4", true},
		{4, "score", "int8", false},
		{5, "rating", "float4", true},
		{6, "balance", "numeric", true},
		{7, "is_active", "bool", true},
		{8, "data", "jsonb", false},
		{9, "avatar", "bytea", false},
		{10, "created_at", "timestamp", true},
		{11, "updated_at", "timestamptz", false},
		{12, "small_id", "int2", true},
		{13, "auto_id", "serial", true},
		{14, "big_auto", "bigserial", true},
	}

	for _, tt := range tests {
		col := table.Columns[tt.idx]
		if col.Name != tt.name {
			t.Errorf("column[%d]: expected name %q, got %q", tt.idx, tt.name, col.Name)
		}
		if col.Type != tt.typ {
			t.Errorf("column %q: expected type %q, got %q", tt.name, tt.typ, col.Type)
		}
		if col.NotNull != tt.notNull {
			t.Errorf("column %q: expected NotNull=%v, got %v", tt.name, tt.notNull, col.NotNull)
		}
	}
}

func TestPostgresCreateEnum(t *testing.T) {
	path := writeTemp(t, `
		CREATE TYPE status AS ENUM ('active', 'inactive', 'pending');
		CREATE TYPE priority AS ENUM ('low', 'medium', 'high', 'critical');
	`)

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	enums := cat.Schemas[0].Enums
	if len(enums) != 2 {
		t.Fatalf("expected 2 enums, got %d", len(enums))
	}

	if enums[0].Name != "status" {
		t.Errorf("expected enum 'status', got %q", enums[0].Name)
	}
	if len(enums[0].Values) != 3 {
		t.Errorf("expected 3 values, got %d", len(enums[0].Values))
	}
	expectedVals := []string{"active", "inactive", "pending"}
	for i, v := range expectedVals {
		if enums[0].Values[i] != v {
			t.Errorf("enum value[%d]: expected %q, got %q", i, v, enums[0].Values[i])
		}
	}

	if enums[1].Name != "priority" {
		t.Errorf("expected enum 'priority', got %q", enums[1].Name)
	}
	if len(enums[1].Values) != 4 {
		t.Errorf("expected 4 values, got %d", len(enums[1].Values))
	}
}

func TestPostgresAlterEnum(t *testing.T) {
	findEnum := func(t *testing.T, cat *catalog.Catalog, schemaName, enumName string) *catalog.Enum {
		t.Helper()
		for _, s := range cat.Schemas {
			if s.Name != schemaName {
				continue
			}
			for _, e := range s.Enums {
				if e.Name == enumName {
					return e
				}
			}
		}
		t.Fatalf("enum %s.%s not found", schemaName, enumName)
		return nil
	}

	assertValues := func(t *testing.T, enum *catalog.Enum, want ...string) {
		t.Helper()
		got := strings.Join(enum.Values, ",")
		expected := strings.Join(want, ",")
		if got != expected {
			t.Errorf("enum %q values: want [%s], got [%s]", enum.Name, expected, got)
		}
	}

	t.Run("add value appends to end", func(t *testing.T) {
		path := writeTemp(t, `
			CREATE TYPE status AS ENUM ('active', 'inactive');
			ALTER TYPE status ADD VALUE 'pending';
		`)
		cat, err := newPostgresParser().ParseSchema([]string{path})
		if err != nil {
			t.Fatalf("ParseSchema error: %v", err)
		}
		assertValues(t, findEnum(t, cat, "public", "status"), "active", "inactive", "pending")
	})

	t.Run("add value if not exists skips duplicate", func(t *testing.T) {
		path := writeTemp(t, `
			CREATE TYPE status AS ENUM ('active', 'inactive');
			ALTER TYPE status ADD VALUE IF NOT EXISTS 'active';
		`)
		cat, err := newPostgresParser().ParseSchema([]string{path})
		if err != nil {
			t.Fatalf("ParseSchema error: %v", err)
		}
		assertValues(t, findEnum(t, cat, "public", "status"), "active", "inactive")
	})

	t.Run("add value before neighbor", func(t *testing.T) {
		path := writeTemp(t, `
			CREATE TYPE status AS ENUM ('active', 'inactive');
			ALTER TYPE status ADD VALUE 'draft' BEFORE 'active';
		`)
		cat, err := newPostgresParser().ParseSchema([]string{path})
		if err != nil {
			t.Fatalf("ParseSchema error: %v", err)
		}
		assertValues(t, findEnum(t, cat, "public", "status"), "draft", "active", "inactive")
	})

	t.Run("add value after neighbor", func(t *testing.T) {
		path := writeTemp(t, `
			CREATE TYPE status AS ENUM ('active', 'inactive', 'pending');
			ALTER TYPE status ADD VALUE 'archived' AFTER 'pending';
			ALTER TYPE status ADD VALUE 'review' AFTER 'active';
		`)
		cat, err := newPostgresParser().ParseSchema([]string{path})
		if err != nil {
			t.Fatalf("ParseSchema error: %v", err)
		}
		assertValues(t, findEnum(t, cat, "public", "status"),
			"active", "review", "inactive", "pending", "archived")
	})

	t.Run("rename value preserves order", func(t *testing.T) {
		path := writeTemp(t, `
			CREATE TYPE status AS ENUM ('active', 'inactive', 'pending');
			ALTER TYPE status RENAME VALUE 'inactive' TO 'disabled';
		`)
		cat, err := newPostgresParser().ParseSchema([]string{path})
		if err != nil {
			t.Fatalf("ParseSchema error: %v", err)
		}
		assertValues(t, findEnum(t, cat, "public", "status"), "active", "disabled", "pending")
	})

	t.Run("schema-qualified alter type", func(t *testing.T) {
		path := writeTemp(t, `
			CREATE SCHEMA IF NOT EXISTS billing;
			CREATE TYPE billing.payment_status AS ENUM ('pending', 'paid');
			ALTER TYPE billing.payment_status ADD VALUE 'refunded';
		`)
		cat, err := newPostgresParser().ParseSchema([]string{path})
		if err != nil {
			t.Fatalf("ParseSchema error: %v", err)
		}
		assertValues(t, findEnum(t, cat, "billing", "payment_status"),
			"pending", "paid", "refunded")
	})
}

func TestPostgresAlterTable(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE profiles (
			id SERIAL PRIMARY KEY,
			user_id UUID NOT NULL
		);
		ALTER TABLE profiles ADD COLUMN bio TEXT NULL;
		ALTER TABLE profiles ADD COLUMN website VARCHAR(255) NOT NULL;
		ALTER TABLE profiles DROP COLUMN website;
		ALTER TABLE profiles ALTER COLUMN bio SET NOT NULL;
		ALTER TABLE profiles ALTER COLUMN bio TYPE VARCHAR(1000);
	`)

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	table := cat.Schemas[0].Tables[0]
	if table.Name != "profiles" {
		t.Fatalf("expected table 'profiles', got %q", table.Name)
	}

	// After all alterations: id, user_id, bio (website added then dropped)
	if len(table.Columns) != 3 {
		t.Fatalf("expected 3 columns after alter, got %d", len(table.Columns))
	}

	// bio should be NOT NULL after SET NOT NULL
	bio := table.Columns[2]
	if bio.Name != "bio" {
		t.Errorf("expected column 'bio', got %q", bio.Name)
	}
	if !bio.NotNull {
		t.Error("bio should be NOT NULL after ALTER SET NOT NULL")
	}
	// bio type changed after ALTER TYPE (pg_query resolves VARCHAR(1000) to varchar)
	if bio.Type != "varchar" && bio.Type != "text" {
		t.Errorf("expected bio type 'varchar' or 'text' after ALTER TYPE, got %q", bio.Type)
	}
}

func TestPostgresRenameTableAndColumn(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE old_name (
			id SERIAL PRIMARY KEY,
			bio TEXT NULL
		);
		ALTER TABLE old_name RENAME TO new_name;
		ALTER TABLE new_name RENAME COLUMN bio TO description;
	`)

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	table := cat.Schemas[0].Tables[0]
	if table.Name != "new_name" {
		t.Fatalf("expected table 'new_name', got %q", table.Name)
	}

	found := false
	for _, col := range table.Columns {
		if col.Name == "description" {
			found = true
		}
		if col.Name == "bio" {
			t.Error("column 'bio' should have been renamed to 'description'")
		}
	}
	if !found {
		t.Error("expected renamed column 'description' not found")
	}
}

func TestPostgresDropTable(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE temp_data (id SERIAL PRIMARY KEY, data TEXT);
		CREATE TABLE keep_me (id SERIAL PRIMARY KEY);
		DROP TABLE temp_data;
	`)

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	tables := cat.Schemas[0].Tables
	if len(tables) != 1 {
		t.Fatalf("expected 1 table after drop, got %d", len(tables))
	}
	if tables[0].Name != "keep_me" {
		t.Errorf("expected 'keep_me', got %q", tables[0].Name)
	}
}

func TestPostgresDropType(t *testing.T) {
	path := writeTemp(t, `
		CREATE TYPE temp_enum AS ENUM ('a', 'b');
		CREATE TYPE keep_enum AS ENUM ('x', 'y');
		DROP TYPE temp_enum;
	`)

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	enums := cat.Schemas[0].Enums
	if len(enums) != 1 {
		t.Fatalf("expected 1 enum after drop, got %d", len(enums))
	}
	if enums[0].Name != "keep_enum" {
		t.Errorf("expected 'keep_enum', got %q", enums[0].Name)
	}
}

func TestPostgresComment(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			email TEXT NOT NULL
		);
		COMMENT ON TABLE users IS 'The main users table';
		COMMENT ON COLUMN users.email IS 'User email address';
	`)

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	table := cat.Schemas[0].Tables[0]
	if table.Comment != "The main users table" {
		t.Errorf("expected table comment, got %q", table.Comment)
	}

	emailCol := table.Columns[1]
	if emailCol.Comment != "User email address" {
		t.Errorf("expected column comment, got %q", emailCol.Comment)
	}
}

func TestPostgresSchema(t *testing.T) {
	path := writeTemp(t, `
		CREATE SCHEMA IF NOT EXISTS custom;
		CREATE TABLE custom.settings (
			id SERIAL PRIMARY KEY,
			key VARCHAR(255) NOT NULL,
			value TEXT NULL
		);
		CREATE TABLE public.users (
			id SERIAL PRIMARY KEY
		);
	`)

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	if len(cat.Schemas) != 2 {
		t.Fatalf("expected 2 schemas, got %d", len(cat.Schemas))
	}

	// Find custom schema
	customSchema, publicSchema := (*struct{ tables int })(nil), (*struct{ tables int })(nil)
	_ = customSchema
	_ = publicSchema

	for _, s := range cat.Schemas {
		switch s.Name {
		case "custom":
			if len(s.Tables) != 1 {
				t.Errorf("custom schema: expected 1 table, got %d", len(s.Tables))
			}
			if s.Tables[0].Name != "settings" {
				t.Errorf("expected table 'settings', got %q", s.Tables[0].Name)
			}
			if s.Tables[0].Schema != "custom" {
				t.Errorf("expected schema 'custom', got %q", s.Tables[0].Schema)
			}
		case "public":
			if len(s.Tables) != 1 {
				t.Errorf("public schema: expected 1 table, got %d", len(s.Tables))
			}
		}
	}
}

func TestPostgresIfNotExists(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL
		);
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			different_column TEXT
		);
	`)

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	tables := cat.Schemas[0].Tables
	if len(tables) != 1 {
		t.Fatalf("expected 1 table (duplicate ignored), got %d", len(tables))
	}
	// Should have original columns, not the duplicate's
	if len(tables[0].Columns) != 2 {
		t.Errorf("expected 2 columns from original, got %d", len(tables[0].Columns))
	}
	if tables[0].Columns[1].Name != "name" {
		t.Errorf("expected column 'name', got %q", tables[0].Columns[1].Name)
	}
}

func TestPostgresArrayColumns(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE data (
			id SERIAL PRIMARY KEY,
			tags TEXT[] NOT NULL,
			matrix INTEGER[][] NULL
		);
	`)

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	table := cat.Schemas[0].Tables[0]

	tags := table.Columns[1]
	if !tags.IsArray {
		t.Error("tags should be an array")
	}
	if tags.ArrayDims != 1 {
		t.Errorf("tags expected ArrayDims=1, got %d", tags.ArrayDims)
	}

	matrix := table.Columns[2]
	if !matrix.IsArray {
		t.Error("matrix should be an array")
	}
	if matrix.ArrayDims != 2 {
		t.Errorf("matrix expected ArrayDims=2, got %d", matrix.ArrayDims)
	}
}

func TestPostgresFullMigrationDirectory(t *testing.T) {
	migrationDir := filepath.Join("..", "..", "testdata", "sql", "migrations", "postgres")
	if _, err := os.Stat(migrationDir); os.IsNotExist(err) {
		t.Skip("migration directory not found, skipping")
	}

	files, err := ResolveSchemaFiles(migrationDir)
	if err != nil {
		t.Fatalf("ResolveSchemaFiles error: %v", err)
	}

	// Verify no .down.sql files are included
	for _, f := range files {
		if strings.HasSuffix(strings.ToLower(f), ".down.sql") {
			t.Errorf("unexpected .down.sql file: %s", f)
		}
	}

	p := newPostgresParser()
	cat, err := p.ParseSchema(files)
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	schema := cat.Schemas[0]

	// Verify key tables exist from the migration
	requiredTables := []string{
		"authors",
		"books",
	}

	tableNames := make(map[string]bool)
	for _, tbl := range schema.Tables {
		tableNames[tbl.Name] = true
	}

	for _, name := range requiredTables {
		if !tableNames[name] {
			t.Errorf("table %q not found in parsed schema", name)
		}
	}

	// Verify books table has expected columns
	for _, tbl := range schema.Tables {
		if tbl.Name == "books" {
			colMap := make(map[string]*catalog.Column)
			for _, c := range tbl.Columns {
				colMap[c.Name] = c
			}
			// Check genre column exists with enum type
			if c, ok := colMap["genre"]; !ok {
				t.Error("books: missing column \"genre\"")
			} else if !c.NotNull {
				t.Error("books.genre: expected NotNull=true")
			}
			// Check author_id column exists
			if _, ok := colMap["author_id"]; !ok {
				t.Error("books: missing column \"author_id\"")
			}
			break
		}
	}

	// Verify book_type enum was parsed
	if len(schema.Enums) == 0 {
		t.Error("expected at least one enum (book_type)")
	} else {
		found := false
		for _, e := range schema.Enums {
			if e.Name == "book_type" {
				found = true
				if len(e.Values) != 3 {
					t.Errorf("book_type: expected 3 values, got %d", len(e.Values))
				}
			}
		}
		if !found {
			t.Error("enum \"book_type\" not found in parsed schema")
		}
	}

	t.Logf("parsed %d tables from %d migration files", len(schema.Tables), len(files))
}

func TestPostgresRealMigration(t *testing.T) {
	// Use the existing testdata migration file
	migrationPath := filepath.Join("..", "..", "testdata", "sql", "migrations", "postgres", "000001_init_ddl.up.sql")
	if _, err := os.Stat(migrationPath); os.IsNotExist(err) {
		t.Skip("migration file not found, skipping")
	}

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{migrationPath})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	schema := cat.Schemas[0]

	// Should have 2 tables: authors and books
	if len(schema.Tables) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(schema.Tables))
	}

	// Check authors table
	authors, books := (*struct{})(nil), (*struct{})(nil)
	_ = authors
	_ = books

	for _, table := range schema.Tables {
		switch table.Name {
		case "authors":
			if len(table.Columns) != 9 {
				t.Errorf("authors: expected 9 columns, got %d", len(table.Columns))
			}
			// Check id column
			if table.Columns[0].Name != "id" || table.Columns[0].Type != "uuid" || !table.Columns[0].NotNull {
				t.Errorf("authors.id: unexpected %+v", table.Columns[0])
			}
			// Check jsonb column
			found := false
			for _, c := range table.Columns {
				if c.Name == "notifications" && c.Type == "jsonb" && c.NotNull {
					found = true
				}
			}
			if !found {
				t.Error("authors: missing notifications JSONB NOT NULL column")
			}
		case "books":
			if len(table.Columns) != 8 {
				t.Errorf("books: expected 8 columns, got %d", len(table.Columns))
			}
			// Check genre column (enum type)
			found := false
			for _, c := range table.Columns {
				if c.Name == "genre" && c.Type == "book_type" && c.NotNull {
					found = true
				}
			}
			if !found {
				t.Error("books: missing genre book_type NOT NULL column")
			}
		}
	}

	// Should have book_type enum (DROP TYPE book_type runs first, then CREATE TYPE)
	if len(schema.Enums) != 1 {
		t.Fatalf("expected 1 enum, got %d", len(schema.Enums))
	}
	if schema.Enums[0].Name != "book_type" {
		t.Errorf("expected enum 'book_type', got %q", schema.Enums[0].Name)
	}
	if len(schema.Enums[0].Values) != 3 {
		t.Errorf("expected 3 enum values, got %d", len(schema.Enums[0].Values))
	}
}

func TestPostgresMultipleFiles(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "001.sql")
	if err := os.WriteFile(file1, []byte(`CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL);`), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	file2 := filepath.Join(dir, "002.sql")
	if err := os.WriteFile(file2, []byte(`CREATE TABLE posts (id SERIAL PRIMARY KEY, user_id INTEGER NOT NULL);`), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{file1, file2})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	if len(cat.Schemas[0].Tables) != 2 {
		t.Fatalf("expected 2 tables from 2 files, got %d", len(cat.Schemas[0].Tables))
	}
}

func TestPostgresDropNotNull(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE accounts (
			id SERIAL PRIMARY KEY,
			email TEXT NOT NULL,
			phone TEXT NOT NULL
		);
		ALTER TABLE accounts ALTER COLUMN phone DROP NOT NULL;
	`)

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	table := cat.Schemas[0].Tables[0]
	for _, col := range table.Columns {
		switch col.Name {
		case "email":
			if !col.NotNull {
				t.Error("email should remain NOT NULL")
			}
		case "phone":
			if col.NotNull {
				t.Error("phone should be nullable after DROP NOT NULL")
			}
		}
	}
}

func TestPostgresColumnDefaults(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE users (
			id UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
			is_active BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			status TEXT NOT NULL DEFAULT 'pending'
		);
	`)

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	table := cat.Schemas[0].Tables[0]
	defaults := map[string]string{
		"id":         "uuid_generate_v4()",
		"is_active":  "false",
		"created_at": "current_timestamp",
		"status":     "'pending'",
	}
	for _, col := range table.Columns {
		if expected, ok := defaults[col.Name]; ok {
			if !strings.EqualFold(col.Default, expected) {
				t.Errorf("column %q: expected default %q, got %q", col.Name, expected, col.Default)
			}
		}
	}
}

func TestPostgresPrimaryKey(t *testing.T) {
	t.Run("inline", func(t *testing.T) {
		path := writeTemp(t, `
			CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name TEXT NOT NULL
			);
		`)
		p := newPostgresParser()
		cat, err := p.ParseSchema([]string{path})
		if err != nil {
			t.Fatalf("ParseSchema error: %v", err)
		}
		table := cat.Schemas[0].Tables[0]
		if table.PrimaryKey == nil {
			t.Fatal("expected PrimaryKey to be set")
		}
		if len(table.PrimaryKey.Columns) != 1 || table.PrimaryKey.Columns[0] != "id" {
			t.Errorf("expected PK columns [id], got %v", table.PrimaryKey.Columns)
		}
		if !table.Columns[0].IsPrimary {
			t.Error("id column should have IsPrimary=true")
		}
	})

	t.Run("composite", func(t *testing.T) {
		path := writeTemp(t, `
			CREATE TABLE order_items (
				order_id INTEGER NOT NULL,
				product_id INTEGER NOT NULL,
				quantity INTEGER NOT NULL,
				PRIMARY KEY (order_id, product_id)
			);
		`)
		p := newPostgresParser()
		cat, err := p.ParseSchema([]string{path})
		if err != nil {
			t.Fatalf("ParseSchema error: %v", err)
		}
		table := cat.Schemas[0].Tables[0]
		if table.PrimaryKey == nil {
			t.Fatal("expected PrimaryKey to be set")
		}
		if len(table.PrimaryKey.Columns) != 2 {
			t.Fatalf("expected 2 PK columns, got %d", len(table.PrimaryKey.Columns))
		}
		if table.PrimaryKey.Columns[0] != "order_id" || table.PrimaryKey.Columns[1] != "product_id" {
			t.Errorf("expected PK [order_id, product_id], got %v", table.PrimaryKey.Columns)
		}
	})
}

func TestPostgresForeignKey(t *testing.T) {
	t.Run("inline_references", func(t *testing.T) {
		path := writeTemp(t, `
			CREATE TABLE authors (id UUID PRIMARY KEY);
			CREATE TABLE books (
				id UUID PRIMARY KEY,
				author_id UUID NOT NULL REFERENCES authors (id)
			);
		`)
		p := newPostgresParser()
		cat, err := p.ParseSchema([]string{path})
		if err != nil {
			t.Fatalf("ParseSchema error: %v", err)
		}
		books := cat.Schemas[0].Tables[1]
		if len(books.ForeignKeys) != 1 {
			t.Fatalf("expected 1 FK, got %d", len(books.ForeignKeys))
		}
		fk := books.ForeignKeys[0]
		if fk.RefTable != "authors" {
			t.Errorf("expected ref table 'authors', got %q", fk.RefTable)
		}
		if len(fk.Columns) != 1 || fk.Columns[0] != "author_id" {
			t.Errorf("expected FK columns [author_id], got %v", fk.Columns)
		}
		if len(fk.RefColumns) != 1 || fk.RefColumns[0] != "id" {
			t.Errorf("expected ref columns [id], got %v", fk.RefColumns)
		}
	})

	t.Run("table_level", func(t *testing.T) {
		path := writeTemp(t, `
			CREATE TABLE parents (id UUID PRIMARY KEY);
			CREATE TABLE children (
				id UUID PRIMARY KEY,
				parent_id UUID NOT NULL,
				CONSTRAINT fk_parent FOREIGN KEY (parent_id) REFERENCES parents (id) ON DELETE CASCADE
			);
		`)
		p := newPostgresParser()
		cat, err := p.ParseSchema([]string{path})
		if err != nil {
			t.Fatalf("ParseSchema error: %v", err)
		}
		children := cat.Schemas[0].Tables[1]
		if len(children.ForeignKeys) != 1 {
			t.Fatalf("expected 1 FK, got %d", len(children.ForeignKeys))
		}
		fk := children.ForeignKeys[0]
		if fk.Name != "fk_parent" {
			t.Errorf("expected FK name 'fk_parent', got %q", fk.Name)
		}
		if fk.OnDelete != "CASCADE" {
			t.Errorf("expected ON DELETE CASCADE, got %q", fk.OnDelete)
		}
	})
}

func TestPostgresUniqueConstraint(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			email TEXT NOT NULL,
			username TEXT NOT NULL,
			UNIQUE (email),
			CONSTRAINT uq_username UNIQUE (username)
		);
	`)
	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}
	table := cat.Schemas[0].Tables[0]
	if len(table.Uniques) != 2 {
		t.Fatalf("expected 2 unique constraints, got %d", len(table.Uniques))
	}
	if table.Uniques[1].Name != "uq_username" {
		t.Errorf("expected constraint name 'uq_username', got %q", table.Uniques[1].Name)
	}
}

func TestPostgresCheckConstraint(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			price NUMERIC NOT NULL,
			CONSTRAINT positive_price CHECK (price > 0)
		);
	`)
	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}
	table := cat.Schemas[0].Tables[0]
	if len(table.Checks) != 1 {
		t.Fatalf("expected 1 check constraint, got %d", len(table.Checks))
	}
	if table.Checks[0].Name != "positive_price" {
		t.Errorf("expected name 'positive_price', got %q", table.Checks[0].Name)
	}
	if table.Checks[0].Expression == "" {
		t.Error("expected check expression to be set")
	}
}

func TestPostgresCreateIndex(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			email TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_email ON users (email) WHERE email != '';
		CREATE INDEX idx_created ON users (created_at);
	`)
	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}
	table := cat.Schemas[0].Tables[0]
	if len(table.Indexes) != 2 {
		t.Fatalf("expected 2 indexes, got %d", len(table.Indexes))
	}

	idx0 := table.Indexes[0]
	if idx0.Name != "idx_email" {
		t.Errorf("expected name 'idx_email', got %q", idx0.Name)
	}
	if !idx0.IsUnique {
		t.Error("idx_email should be unique")
	}
	if !idx0.IfNotExists {
		t.Error("idx_email should have IfNotExists")
	}
	if idx0.Where == "" {
		t.Error("idx_email should have WHERE clause")
	}

	idx1 := table.Indexes[1]
	if idx1.Name != "idx_created" {
		t.Errorf("expected name 'idx_created', got %q", idx1.Name)
	}
	if idx1.IsUnique {
		t.Error("idx_created should not be unique")
	}
}

func TestPostgresCreateExtension(t *testing.T) {
	path := writeTemp(t, `
		CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
		CREATE EXTENSION IF NOT EXISTS "pgcrypto";
	`)
	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}
	exts := cat.Schemas[0].Extensions
	if len(exts) != 2 {
		t.Fatalf("expected 2 extensions, got %d", len(exts))
	}
	if exts[0] != "uuid-ossp" || exts[1] != "pgcrypto" {
		t.Errorf("expected [uuid-ossp, pgcrypto], got %v", exts)
	}
}

func TestPostgresDropIndex(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE users (id SERIAL PRIMARY KEY, email TEXT NOT NULL);
		CREATE INDEX idx_email ON users (email);
		CREATE INDEX idx_keep ON users (id);
		DROP INDEX idx_email;
	`)
	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}
	table := cat.Schemas[0].Tables[0]
	if len(table.Indexes) != 1 {
		t.Fatalf("expected 1 index after drop, got %d", len(table.Indexes))
	}
	if table.Indexes[0].Name != "idx_keep" {
		t.Errorf("expected 'idx_keep', got %q", table.Indexes[0].Name)
	}
}

func TestPostgresAlterTableSetDefault(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			status TEXT NOT NULL
		);
		ALTER TABLE users ALTER COLUMN status SET DEFAULT 'active';
	`)
	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}
	table := cat.Schemas[0].Tables[0]
	for _, col := range table.Columns {
		if col.Name == "status" {
			if col.Default != "'active'" {
				t.Errorf("expected default 'active', got %q", col.Default)
			}
			return
		}
	}
	t.Error("status column not found")
}

func TestPostgresAlterTableAddConstraint(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE orders (id SERIAL PRIMARY KEY, total NUMERIC NOT NULL);
		ALTER TABLE orders ADD CONSTRAINT positive_total CHECK (total >= 0);
	`)
	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}
	table := cat.Schemas[0].Tables[0]
	if len(table.Checks) != 1 {
		t.Fatalf("expected 1 check constraint, got %d", len(table.Checks))
	}
	if table.Checks[0].Name != "positive_total" {
		t.Errorf("expected 'positive_total', got %q", table.Checks[0].Name)
	}
}

func TestPostgresAlterTableDropConstraint(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE orders (
			id SERIAL PRIMARY KEY,
			total NUMERIC NOT NULL,
			CONSTRAINT positive_total CHECK (total >= 0)
		);
		ALTER TABLE orders DROP CONSTRAINT positive_total;
	`)
	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}
	table := cat.Schemas[0].Tables[0]
	if len(table.Checks) != 0 {
		t.Errorf("expected 0 check constraints after drop, got %d", len(table.Checks))
	}
}

func TestPostgresFullType(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE data (
			id UUID PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			balance NUMERIC(10,2) NOT NULL,
			data TEXT NOT NULL
		);
	`)
	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}
	table := cat.Schemas[0].Tables[0]
	for _, col := range table.Columns {
		switch col.Name {
		case "name":
			if !strings.Contains(col.FullType, "255") {
				t.Errorf("name FullType should contain '255', got %q", col.FullType)
			}
		case "balance":
			if !strings.Contains(col.FullType, "10") {
				t.Errorf("balance FullType should contain '10', got %q", col.FullType)
			}
		}
	}
}

func TestPostgresParseError(t *testing.T) {
	path := writeTemp(t, `THIS IS NOT VALID SQL AT ALL ???`)

	p := newPostgresParser()
	_, err := p.ParseSchema([]string{path})
	if err == nil {
		t.Fatal("expected parse error for invalid SQL")
	}
}

func TestPostgresEmptyFile(t *testing.T) {
	path := writeTemp(t, ``)

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	if len(cat.Schemas) != 1 {
		t.Fatalf("expected 1 schema (public), got %d", len(cat.Schemas))
	}
	if len(cat.Schemas[0].Tables) != 0 {
		t.Errorf("expected 0 tables, got %d", len(cat.Schemas[0].Tables))
	}
	if len(cat.Schemas[0].Enums) != 0 {
		t.Errorf("expected 0 enums, got %d", len(cat.Schemas[0].Enums))
	}
}

func TestPostgresMultiSchemaEnums(t *testing.T) {
	path := writeTemp(t, `
		CREATE SCHEMA IF NOT EXISTS billing;
		CREATE TYPE status AS ENUM ('active', 'inactive');
		CREATE TYPE billing.payment_status AS ENUM ('pending', 'paid', 'refunded');
	`)

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	if len(cat.Schemas) != 2 {
		t.Fatalf("expected 2 schemas, got %d", len(cat.Schemas))
	}

	for _, s := range cat.Schemas {
		switch s.Name {
		case "public":
			if len(s.Enums) != 1 || s.Enums[0].Name != "status" {
				t.Errorf("public: expected enum 'status', got %v", s.Enums)
			}
		case "billing":
			if len(s.Enums) != 1 || s.Enums[0].Name != "payment_status" {
				t.Errorf("billing: expected enum 'payment_status', got %v", s.Enums)
			}
			if len(s.Enums) == 1 && len(s.Enums[0].Values) != 3 {
				t.Errorf("billing.payment_status: expected 3 values, got %d", len(s.Enums[0].Values))
			}
		}
	}
}

func TestPostgresTestdataFile(t *testing.T) {
	path := filepath.Join("testdata", "postgresql.sql")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("testdata file not found")
	}

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	// public schema should have: users, posts, profiles (temp_data dropped)
	var publicSchema *struct{}
	_ = publicSchema
	for _, s := range cat.Schemas {
		if s.Name == "public" {
			if len(s.Tables) != 3 {
				t.Errorf("public schema: expected 3 tables (users, posts, profiles), got %d", len(s.Tables))
				for _, tbl := range s.Tables {
					t.Logf("  table: %s", tbl.Name)
				}
			}
			// status enum should exist, priority should be dropped
			if len(s.Enums) != 1 {
				t.Errorf("expected 1 enum (status, priority dropped), got %d", len(s.Enums))
			}
			if len(s.Enums) > 0 && s.Enums[0].Name != "status" {
				t.Errorf("expected enum 'status', got %q", s.Enums[0].Name)
			}
		}
		if s.Name == "custom_schema" {
			if len(s.Tables) != 1 {
				t.Errorf("custom_schema: expected 1 table, got %d", len(s.Tables))
			}
		}
	}
}

// --- View tests ---

func TestPostgresCreateView(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE users (
			id UUID NOT NULL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email TEXT,
			created_at TIMESTAMP NOT NULL
		);

		CREATE VIEW active_users AS
			SELECT id, name, email FROM users;
	`)

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	schema := cat.Schemas[0]
	if len(schema.Views) != 1 {
		t.Fatalf("expected 1 view, got %d", len(schema.Views))
	}

	view := schema.Views[0]
	if view.Name != "active_users" {
		t.Errorf("expected view name 'active_users', got %q", view.Name)
	}
	if view.Schema != "public" {
		t.Errorf("expected schema 'public', got %q", view.Schema)
	}

	if len(view.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(view.Columns))
	}

	// Verify column names and types resolved from base table
	expectedCols := []struct {
		name string
		typ  string
	}{
		{"id", "uuid"},
		{"name", "varchar"},
		{"email", "text"},
	}
	for i, ec := range expectedCols {
		if view.Columns[i].Name != ec.name {
			t.Errorf("column %d: expected name %q, got %q", i, ec.name, view.Columns[i].Name)
		}
		if !strings.EqualFold(view.Columns[i].Type, ec.typ) {
			t.Errorf("column %d (%s): expected type %q, got %q", i, ec.name, ec.typ, view.Columns[i].Type)
		}
	}

	if view.Query == "" {
		t.Error("expected non-empty query")
	}
}

func TestPostgresCreateViewWithAliases(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE products (
			id SERIAL NOT NULL,
			title TEXT NOT NULL,
			price NUMERIC(10,2) NOT NULL
		);

		CREATE VIEW product_summary (product_id, product_title) AS
			SELECT id, title FROM products;
	`)

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	schema := cat.Schemas[0]
	if len(schema.Views) != 1 {
		t.Fatalf("expected 1 view, got %d", len(schema.Views))
	}

	view := schema.Views[0]
	if len(view.Columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(view.Columns))
	}

	// Explicit aliases should override column names
	if view.Columns[0].Name != "product_id" {
		t.Errorf("expected column name 'product_id', got %q", view.Columns[0].Name)
	}
	if view.Columns[1].Name != "product_title" {
		t.Errorf("expected column name 'product_title', got %q", view.Columns[1].Name)
	}
}

func TestPostgresCreateViewStar(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE orders (
			id UUID NOT NULL,
			amount NUMERIC(10,2) NOT NULL,
			status TEXT NOT NULL
		);

		CREATE VIEW all_orders AS SELECT * FROM orders;
	`)

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	schema := cat.Schemas[0]
	if len(schema.Views) != 1 {
		t.Fatalf("expected 1 view, got %d", len(schema.Views))
	}

	view := schema.Views[0]
	if len(view.Columns) != 3 {
		t.Fatalf("expected 3 columns from SELECT *, got %d", len(view.Columns))
	}

	if view.Columns[0].Name != "id" {
		t.Errorf("expected first column 'id', got %q", view.Columns[0].Name)
	}
}

func TestPostgresDropView(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE t1 (id INT NOT NULL);
		CREATE VIEW v1 AS SELECT id FROM t1;
		DROP VIEW v1;
	`)

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	if len(cat.Schemas[0].Views) != 0 {
		t.Errorf("expected 0 views after DROP, got %d", len(cat.Schemas[0].Views))
	}
}

func TestPostgresCommentOnView(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE t1 (id INT NOT NULL);
		CREATE VIEW v1 AS SELECT id FROM t1;
		COMMENT ON VIEW v1 IS 'my view comment';
	`)

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	if len(cat.Schemas[0].Views) != 1 {
		t.Fatalf("expected 1 view, got %d", len(cat.Schemas[0].Views))
	}
	if cat.Schemas[0].Views[0].Comment != "my view comment" {
		t.Errorf("expected comment 'my view comment', got %q", cat.Schemas[0].Views[0].Comment)
	}
}

func TestPostgresCreateOrReplaceView(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE t1 (id INT NOT NULL, name TEXT NOT NULL);
		CREATE VIEW v1 AS SELECT id FROM t1;
		CREATE OR REPLACE VIEW v1 AS SELECT id, name FROM t1;
	`)

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	if len(cat.Schemas[0].Views) != 1 {
		t.Fatalf("expected 1 view, got %d", len(cat.Schemas[0].Views))
	}
	if len(cat.Schemas[0].Views[0].Columns) != 2 {
		t.Errorf("expected 2 columns after replace, got %d", len(cat.Schemas[0].Views[0].Columns))
	}
}

func TestPostgresCreateViewWithJoin(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE webhooks (
			id UUID NOT NULL PRIMARY KEY,
			kind TEXT NOT NULL,
			status TEXT NOT NULL,
			attempts INT NOT NULL DEFAULT 0,
			payload BYTEA NOT NULL,
			client_id UUID NOT NULL,
			response TEXT,
			created_at TIMESTAMPTZ,
			sent_at TIMESTAMPTZ,
			updated_at TIMESTAMPTZ
		);

		CREATE TABLE clients (
			id UUID NOT NULL PRIMARY KEY,
			name TEXT NOT NULL,
			callback_url TEXT NOT NULL,
			secret_key TEXT NOT NULL
		);

		CREATE OR REPLACE VIEW webhook_view AS (
			SELECT w.*, c.callback_url, c.secret_key
			FROM webhooks w
			JOIN clients c ON c.id = w.client_id
		);
	`)

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	schema := cat.Schemas[0]
	if len(schema.Views) != 1 {
		t.Fatalf("expected 1 view, got %d", len(schema.Views))
	}

	view := schema.Views[0]
	if view.Name != "webhook_view" {
		t.Errorf("expected view name 'webhook_view', got %q", view.Name)
	}

	// w.* should expand to all 10 webhooks columns + 2 explicit columns from clients = 12
	if len(view.Columns) != 12 {
		t.Fatalf("expected 12 columns, got %d", len(view.Columns))
		for _, col := range view.Columns {
			t.Logf("  %s: %s", col.Name, col.Type)
		}
	}

	// Verify first column (from w.*)
	if view.Columns[0].Name != "id" {
		t.Errorf("expected first column 'id', got %q", view.Columns[0].Name)
	}
	if !strings.EqualFold(view.Columns[0].Type, "uuid") {
		t.Errorf("expected type 'uuid' for id, got %q", view.Columns[0].Type)
	}

	// Verify last two columns (from c.callback_url, c.secret_key)
	callbackCol := view.Columns[10]
	if callbackCol.Name != "callback_url" {
		t.Errorf("expected column 'callback_url', got %q", callbackCol.Name)
	}
	if !strings.EqualFold(callbackCol.Type, "text") {
		t.Errorf("expected type 'text' for callback_url, got %q", callbackCol.Type)
	}

	secretCol := view.Columns[11]
	if secretCol.Name != "secret_key" {
		t.Errorf("expected column 'secret_key', got %q", secretCol.Name)
	}
	if !strings.EqualFold(secretCol.Type, "text") {
		t.Errorf("expected type 'text' for secret_key, got %q", secretCol.Type)
	}
}

func TestPostgresCreateViewInfersBuiltinFunctionTypes(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE products (
			id UUID NOT NULL PRIMARY KEY,
			name TEXT NOT NULL,
			quantity INTEGER NOT NULL,
			price NUMERIC(10,2),
			published_at TIMESTAMPTZ NOT NULL,
			categories TEXT[] NOT NULL
		);

		CREATE TABLE product_reviews (
			id UUID NOT NULL PRIMARY KEY,
			product_id UUID NOT NULL REFERENCES products (id)
		);

		CREATE VIEW product_statistics AS
			SELECT
				count(*) AS product_count,
				sum(quantity) AS quantity_sum,
				avg(quantity) AS quantity_average,
				min(quantity) AS minimum_quantity,
				max(quantity) AS maximum_quantity,
				bool_and(quantity > 0) AS all_available,
				bit_and(quantity) AS quantity_bits,
				corr(price, price) AS price_correlation,
				regr_count(price, price) AS regression_count,
				stddev(quantity) AS quantity_deviation,
				string_agg(name, ',') AS product_names,
				json_agg(name) AS product_names_json,
				jsonb_agg(name) AS product_names_jsonb,
				array_agg(quantity) AS quantities
			FROM products;

		CREATE VIEW product_details AS
			SELECT
				lower(name) AS normalized_name,
				pg_catalog.upper(name) AS uppercase_name,
				char_length(name) AS name_length,
				concat(name, quantity) AS display_name,
				format('%s: %s', name, quantity) AS formatted_name,
				concat_ws(':', name, quantity) AS joined_name,
				date_trunc('day', published_at) AS publication_day,
				date_part('epoch', published_at) AS publication_epoch,
				to_date('2026-01-01', 'YYYY-MM-DD') AS parsed_date,
				to_timestamp(0) AS unix_epoch,
				coalesce(price, 0::numeric) AS effective_price,
				greatest(quantity, 0) AS effective_quantity,
				abs(quantity) AS absolute_quantity,
				array_append(categories, 'featured') AS extended_categories,
				json_build_object('name', name) AS product_json,
				jsonb_build_object('name', name) AS product_jsonb,
				to_json(name) AS name_json,
				to_jsonb(name) AS name_jsonb,
				gen_random_uuid() AS generated_id,
				CURRENT_TIMESTAMP AS generated_at,
				CURRENT_DATE AS generated_date,
				CURRENT_TIME AS generated_time,
				LOCALTIMESTAMP AS generated_local_timestamp,
				CURRENT_USER AS generated_by,
				now() AS current_instant,
				timeofday() AS current_time_text,
				random() AS random_value,
				pg_backend_pid() AS backend_pid,
				current_schema() AS schema_name,
				current_schemas(true) AS schema_names,
				row_number() OVER (ORDER BY id) AS row_position,
				percent_rank() OVER (ORDER BY quantity) AS quantity_rank,
				ntile(4) OVER (ORDER BY quantity) AS quantity_bucket,
				lag(name) OVER (ORDER BY id) AS previous_name
			FROM products;

		CREATE VIEW product_review_counts AS
			SELECT
				p.id,
				(SELECT count(*) FROM product_reviews r WHERE r.product_id = p.id) AS review_count
			FROM products p;
	`)

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	tests := []struct {
		view     string
		column   string
		wantType string
		wantGo   string
		notNull  bool
		isArray  bool
	}{
		{view: "product_statistics", column: "product_count", wantType: "bigint", wantGo: "int64", notNull: true},
		{view: "product_statistics", column: "quantity_sum", wantType: "bigint", wantGo: "pgtype.Int8"},
		{view: "product_statistics", column: "quantity_average", wantType: "numeric", wantGo: "pgtype.Numeric"},
		{view: "product_statistics", column: "minimum_quantity", wantType: "int4", wantGo: "pgtype.Int4"},
		{view: "product_statistics", column: "maximum_quantity", wantType: "int4", wantGo: "pgtype.Int4"},
		{view: "product_statistics", column: "all_available", wantType: "boolean", wantGo: "pgtype.Bool"},
		{view: "product_statistics", column: "quantity_bits", wantType: "int4", wantGo: "pgtype.Int4"},
		{view: "product_statistics", column: "price_correlation", wantType: "double precision", wantGo: "pgtype.Float8"},
		{view: "product_statistics", column: "regression_count", wantType: "bigint", wantGo: "int64", notNull: true},
		{view: "product_statistics", column: "quantity_deviation", wantType: "numeric", wantGo: "pgtype.Numeric"},
		{view: "product_statistics", column: "product_names", wantType: "text", wantGo: "pgtype.Text"},
		{view: "product_statistics", column: "product_names_json", wantType: "json", wantGo: "[]byte"},
		{view: "product_statistics", column: "product_names_jsonb", wantType: "jsonb", wantGo: "[]byte"},
		{view: "product_statistics", column: "quantities", wantType: "int4", wantGo: "[]int32", isArray: true},
		{view: "product_details", column: "normalized_name", wantType: "text", wantGo: "string", notNull: true},
		{view: "product_details", column: "uppercase_name", wantType: "text", wantGo: "string", notNull: true},
		{view: "product_details", column: "name_length", wantType: "integer", wantGo: "int32", notNull: true},
		{view: "product_details", column: "display_name", wantType: "text", wantGo: "string", notNull: true},
		{view: "product_details", column: "formatted_name", wantType: "text", wantGo: "string", notNull: true},
		{view: "product_details", column: "joined_name", wantType: "text", wantGo: "string", notNull: true},
		{view: "product_details", column: "publication_day", wantType: "timestamptz", wantGo: "pgtype.Timestamptz", notNull: true},
		{view: "product_details", column: "publication_epoch", wantType: "double precision", wantGo: "float64", notNull: true},
		{view: "product_details", column: "parsed_date", wantType: "date", wantGo: "pgtype.Date", notNull: true},
		{view: "product_details", column: "unix_epoch", wantType: "timestamptz", wantGo: "pgtype.Timestamptz", notNull: true},
		{view: "product_details", column: "effective_price", wantType: "numeric", wantGo: "pgtype.Numeric", notNull: true},
		{view: "product_details", column: "effective_quantity", wantType: "int4", wantGo: "int32", notNull: true},
		{view: "product_details", column: "absolute_quantity", wantType: "int4", wantGo: "int32", notNull: true},
		{view: "product_details", column: "extended_categories", wantType: "text", wantGo: "[]string", notNull: true, isArray: true},
		{view: "product_details", column: "product_json", wantType: "json", wantGo: "[]byte", notNull: true},
		{view: "product_details", column: "product_jsonb", wantType: "jsonb", wantGo: "[]byte", notNull: true},
		{view: "product_details", column: "name_json", wantType: "json", wantGo: "[]byte", notNull: true},
		{view: "product_details", column: "name_jsonb", wantType: "jsonb", wantGo: "[]byte", notNull: true},
		{view: "product_details", column: "generated_id", wantType: "uuid", wantGo: "pgtype.UUID", notNull: true},
		{view: "product_details", column: "generated_at", wantType: "timestamptz", wantGo: "pgtype.Timestamptz", notNull: true},
		{view: "product_details", column: "generated_date", wantType: "date", wantGo: "pgtype.Date", notNull: true},
		{view: "product_details", column: "generated_time", wantType: "pg_catalog.timetz", wantGo: "time.Time", notNull: true},
		{view: "product_details", column: "generated_local_timestamp", wantType: "timestamp", wantGo: "pgtype.Timestamp", notNull: true},
		{view: "product_details", column: "generated_by", wantType: "text", wantGo: "string", notNull: true},
		{view: "product_details", column: "current_instant", wantType: "timestamptz", wantGo: "pgtype.Timestamptz", notNull: true},
		{view: "product_details", column: "current_time_text", wantType: "text", wantGo: "string", notNull: true},
		{view: "product_details", column: "random_value", wantType: "double precision", wantGo: "float64", notNull: true},
		{view: "product_details", column: "backend_pid", wantType: "integer", wantGo: "int32", notNull: true},
		{view: "product_details", column: "schema_name", wantType: "text", wantGo: "pgtype.Text"},
		{view: "product_details", column: "schema_names", wantType: "text", wantGo: "[]string", notNull: true, isArray: true},
		{view: "product_details", column: "row_position", wantType: "bigint", wantGo: "int64", notNull: true},
		{view: "product_details", column: "quantity_rank", wantType: "double precision", wantGo: "float64", notNull: true},
		{view: "product_details", column: "quantity_bucket", wantType: "integer", wantGo: "int32", notNull: true},
		{view: "product_details", column: "previous_name", wantType: "text", wantGo: "pgtype.Text"},
		{view: "product_review_counts", column: "review_count", wantType: "bigint", wantGo: "int64", notNull: true},
	}
	mapper, err := typemap.NewTypeMapper("postgresql")
	if err != nil {
		t.Fatalf("NewTypeMapper error: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.view+"/"+tt.column, func(t *testing.T) {
			var got *catalog.Column
			for _, schema := range cat.Schemas {
				for _, view := range schema.Views {
					if view.Name != tt.view {
						continue
					}
					for _, column := range view.Columns {
						if column.Name == tt.column {
							got = column
							break
						}
					}
				}
			}

			if got == nil {
				t.Fatalf("column %s.%s not found", tt.view, tt.column)
			}
			if got.Type != tt.wantType {
				t.Errorf("type = %q, want %q", got.Type, tt.wantType)
			}
			gotGo := mapper.GoType(got, nil, typemap.Options{SqlPackage: "pgx/v5"})
			if got.IsArray && !strings.HasPrefix(gotGo, "[]") {
				gotGo = "[]" + gotGo
			}
			if gotGo != tt.wantGo {
				t.Errorf("Go type = %q, want %q", gotGo, tt.wantGo)
			}
			if got.NotNull != tt.notNull {
				t.Errorf("not null = %v, want %v", got.NotNull, tt.notNull)
			}
			if got.IsArray != tt.isArray {
				t.Errorf("is array = %v, want %v", got.IsArray, tt.isArray)
			}
		})
	}
}

func TestPostgresNumericAggregateReturnTypes(t *testing.T) {
	tests := []struct {
		name     string
		function string
		argType  string
		want     string
	}{
		{name: "sum smallint", function: "sum", argType: "int2", want: "bigint"},
		{name: "sum integer", function: "sum", argType: "int4", want: "bigint"},
		{name: "sum bigint", function: "sum", argType: "int8", want: "numeric"},
		{name: "sum numeric", function: "sum", argType: "numeric", want: "numeric"},
		{name: "sum real", function: "sum", argType: "float4", want: "real"},
		{name: "sum double precision", function: "sum", argType: "float8", want: "double precision"},
		{name: "sum money", function: "sum", argType: "money", want: "money"},
		{name: "sum interval", function: "sum", argType: "interval", want: "interval"},
		{name: "average integer", function: "avg", argType: "integer", want: "numeric"},
		{name: "average bigint", function: "avg", argType: "bigint", want: "numeric"},
		{name: "average real", function: "avg", argType: "real", want: "double precision"},
		{name: "average interval", function: "avg", argType: "interval", want: "interval"},
		{name: "statistics numeric", function: "statistics", argType: "numeric", want: "numeric"},
		{name: "statistics double precision", function: "statistics", argType: "double precision", want: "double precision"},
		{name: "unsupported sum input", function: "sum", argType: "text", want: "any"},
		{name: "unsupported average input", function: "avg", argType: "text", want: "any"},
		{name: "unsupported statistics input", function: "statistics", argType: "text", want: "any"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got string
			switch tt.function {
			case "sum":
				got = postgresSumType(tt.argType)
			case "avg":
				got = postgresAverageType(tt.argType)
			case "statistics":
				got = postgresStatisticsType(tt.argType)
			default:
				t.Fatalf("unknown aggregate function %q", tt.function)
			}
			if got != tt.want {
				t.Errorf("return type = %q, want %q", got, tt.want)
			}
		})
	}
}
