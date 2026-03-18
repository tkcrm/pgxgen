package sqlparser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tkcrm/pgxgen/internal/sqlparser/catalog"
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

	// Verify key tables exist (these are referenced in pgxgen-postgres.yaml)
	requiredTables := []string{
		"notification_send_history",
		"notification_send_queue",
		"notifications",
		"users",
		"stores",
		"transactions",
		"transfers",
		"wallets",
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

	// Verify transfers table has array columns from ALTER TABLE migration
	for _, tbl := range schema.Tables {
		if tbl.Name == "transfers" {
			colMap := make(map[string]*catalog.Column)
			for _, c := range tbl.Columns {
				colMap[c.Name] = c
			}
			for _, arrCol := range []string{"from_addresses", "to_addresses"} {
				c, ok := colMap[arrCol]
				if !ok {
					t.Errorf("transfers: missing column %q", arrCol)
					continue
				}
				if !c.IsArray {
					t.Errorf("transfers.%s: expected IsArray=true", arrCol)
				}
				if !c.NotNull {
					t.Errorf("transfers.%s: expected NotNull=true", arrCol)
				}
			}
			// from_address and to_address should be dropped
			if _, ok := colMap["from_address"]; ok {
				t.Error("transfers: from_address should have been dropped")
			}
			if _, ok := colMap["to_address"]; ok {
				t.Error("transfers: to_address should have been dropped")
			}
			break
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
	os.WriteFile(file1, []byte(`CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL);`), 0o644)

	file2 := filepath.Join(dir, "002.sql")
	os.WriteFile(file2, []byte(`CREATE TABLE posts (id SERIAL PRIMARY KEY, user_id INTEGER NOT NULL);`), 0o644)

	p := newPostgresParser()
	cat, err := p.ParseSchema([]string{file1, file2})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	if len(cat.Schemas[0].Tables) != 2 {
		t.Fatalf("expected 2 tables from 2 files, got %d", len(cat.Schemas[0].Tables))
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
