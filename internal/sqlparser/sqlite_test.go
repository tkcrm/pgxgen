package sqlparser

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestSqliteCreateTable(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY NOT NULL,
			username TEXT NOT NULL,
			email TEXT NULL,
			age INTEGER NULL,
			is_active BOOLEAN NOT NULL DEFAULT 0,
			balance REAL NOT NULL DEFAULT 0.0,
			metadata TEXT NULL,
			avatar BLOB NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NULL
		);
	`)

	p := newSqliteParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	if cat.DefaultSchema != "main" {
		t.Errorf("expected default schema 'main', got %q", cat.DefaultSchema)
	}

	tables := cat.Schemas[0].Tables
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}

	table := tables[0]
	if table.Name != "users" {
		t.Errorf("expected 'users', got %q", table.Name)
	}

	if len(table.Columns) != 10 {
		t.Fatalf("expected 10 columns, got %d", len(table.Columns))
	}

	// Check specific columns
	tests := []struct {
		idx     int
		name    string
		notNull bool
	}{
		{0, "id", true},       // PRIMARY KEY implies NOT NULL
		{1, "username", true}, // explicit NOT NULL
		{2, "email", false},   // NULL
		{4, "is_active", true},
		{5, "balance", true},
	}

	for _, tt := range tests {
		col := table.Columns[tt.idx]
		if col.Name != tt.name {
			t.Errorf("column[%d]: expected %q, got %q", tt.idx, tt.name, col.Name)
		}
		if col.NotNull != tt.notNull {
			t.Errorf("column %q: expected NotNull=%v, got %v", tt.name, tt.notNull, col.NotNull)
		}
	}
}

func TestSqliteNotNull(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE test (
			pk_col INTEGER PRIMARY KEY,
			notnull_col TEXT NOT NULL,
			nullable_col TEXT,
			pk_notnull INTEGER PRIMARY KEY NOT NULL
		);
	`)

	p := newSqliteParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	table := cat.Schemas[0].Tables[0]

	if !table.Columns[0].NotNull {
		t.Error("PRIMARY KEY column should be NOT NULL")
	}
	if !table.Columns[1].NotNull {
		t.Error("NOT NULL column should be NOT NULL")
	}
	if table.Columns[2].NotNull {
		t.Error("nullable column should not be NOT NULL")
	}
	if !table.Columns[3].NotNull {
		t.Error("PRIMARY KEY NOT NULL should be NOT NULL")
	}
}

func TestSqliteDropTable(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE temp (id INTEGER PRIMARY KEY, data TEXT);
		CREATE TABLE keep (id INTEGER PRIMARY KEY);
		DROP TABLE temp;
	`)

	p := newSqliteParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	tables := cat.Schemas[0].Tables
	if len(tables) != 1 {
		t.Fatalf("expected 1 table after drop, got %d", len(tables))
	}
	if tables[0].Name != "keep" {
		t.Errorf("expected 'keep', got %q", tables[0].Name)
	}
}

func TestSqliteIfNotExists(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
		CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, different TEXT);
	`)

	p := newSqliteParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	tables := cat.Schemas[0].Tables
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
	if len(tables[0].Columns) != 2 {
		t.Errorf("expected 2 columns from original, got %d", len(tables[0].Columns))
	}
	if tables[0].Columns[1].Name != "name" {
		t.Errorf("expected original column 'name', got %q", tables[0].Columns[1].Name)
	}
}

func TestSqliteQuotedIdentifiers(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE "my_table" (
			"my_id" INTEGER PRIMARY KEY NOT NULL,
			"my_name" TEXT NOT NULL
		);
	`)

	p := newSqliteParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	table := cat.Schemas[0].Tables[0]
	if table.Name != "my_table" {
		t.Errorf("expected unquoted 'my_table', got %q", table.Name)
	}
	if table.Columns[0].Name != "my_id" {
		t.Errorf("expected unquoted 'my_id', got %q", table.Columns[0].Name)
	}
}

func TestSqliteMultipleTables(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
		CREATE TABLE posts (id INTEGER PRIMARY KEY, title TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE comments (id INTEGER PRIMARY KEY, body TEXT NOT NULL, post_id INTEGER NOT NULL);
	`)

	p := newSqliteParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	if len(cat.Schemas[0].Tables) != 3 {
		t.Errorf("expected 3 tables, got %d", len(cat.Schemas[0].Tables))
	}
}

func TestSqliteAlterTableAddColumn(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE todos (
			id TEXT PRIMARY KEY NOT NULL,
			user_id TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		ALTER TABLE todos ADD COLUMN workspace_id TEXT;
		ALTER TABLE todos ADD COLUMN priority TEXT NOT NULL DEFAULT 'none';
		ALTER TABLE todos ADD COLUMN recurrence_rule TEXT;
	`)

	p := newSqliteParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	table := cat.Schemas[0].Tables[0]
	if table.Name != "todos" {
		t.Fatalf("expected 'todos', got %q", table.Name)
	}

	// 7 original + 3 from ALTER TABLE ADD COLUMN = 10
	if len(table.Columns) != 10 {
		t.Fatalf("expected 10 columns (7 original + 3 added), got %d", len(table.Columns))
		for _, c := range table.Columns {
			t.Logf("  column: %s", c.Name)
		}
	}

	// Check added columns exist
	colNames := make(map[string]bool)
	for _, c := range table.Columns {
		colNames[c.Name] = true
	}
	for _, expected := range []string{"workspace_id", "priority", "recurrence_rule"} {
		if !colNames[expected] {
			t.Errorf("missing column %q after ALTER TABLE ADD COLUMN", expected)
		}
	}

	// Check priority is NOT NULL
	for _, c := range table.Columns {
		if c.Name == "priority" && !c.NotNull {
			t.Error("priority should be NOT NULL")
		}
		if c.Name == "workspace_id" && c.NotNull {
			t.Error("workspace_id should be nullable")
		}
	}
}

func TestSqliteAlterTableDropColumn(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE users (
			id TEXT PRIMARY KEY NOT NULL,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			legacy_field TEXT
		);
		ALTER TABLE users DROP COLUMN legacy_field;
	`)

	p := newSqliteParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	table := cat.Schemas[0].Tables[0]
	if len(table.Columns) != 3 {
		t.Fatalf("expected 3 columns after DROP COLUMN, got %d", len(table.Columns))
	}
	for _, c := range table.Columns {
		if c.Name == "legacy_field" {
			t.Error("legacy_field should have been dropped")
		}
	}
}

func TestSqliteAlterTableRenameTable(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE old_name (id TEXT PRIMARY KEY NOT NULL);
		ALTER TABLE old_name RENAME TO new_name;
	`)

	p := newSqliteParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	tables := cat.Schemas[0].Tables
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
	if tables[0].Name != "new_name" {
		t.Errorf("expected table renamed to 'new_name', got %q", tables[0].Name)
	}
}

func TestSqliteMultiFileMigrations(t *testing.T) {
	dir := t.TempDir()

	// Migration 1: create tables
	os.WriteFile(filepath.Join(dir, "001.sql"), []byte(`
		CREATE TABLE IF NOT EXISTS todos (
			id TEXT PRIMARY KEY,
			user_id BIGINT NOT NULL,
			title TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS notes (
			id TEXT PRIMARY KEY,
			user_id BIGINT NOT NULL,
			title TEXT NOT NULL,
			body TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`), 0o644)

	// Migration 2: add columns via ALTER TABLE
	os.WriteFile(filepath.Join(dir, "002.sql"), []byte(`
		ALTER TABLE todos ADD COLUMN workspace_id TEXT;
		ALTER TABLE todos ADD COLUMN priority TEXT NOT NULL DEFAULT 'none';
		ALTER TABLE notes ADD COLUMN workspace_id TEXT;
	`), 0o644)

	files, err := ResolveSchemaFiles(dir)
	if err != nil {
		t.Fatalf("ResolveSchemaFiles error: %v", err)
	}

	p := newSqliteParser()
	cat, err := p.ParseSchema(files)
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	// Find todos
	for _, table := range cat.Schemas[0].Tables {
		switch table.Name {
		case "todos":
			// 5 original + 2 added = 7
			if len(table.Columns) != 7 {
				t.Errorf("todos: expected 7 columns (5 + workspace_id + priority), got %d", len(table.Columns))
				for _, c := range table.Columns {
					t.Logf("  column: %s", c.Name)
				}
			}
		case "notes":
			// 5 original + 1 added = 6
			if len(table.Columns) != 6 {
				t.Errorf("notes: expected 6 columns (5 + workspace_id), got %d", len(table.Columns))
				for _, c := range table.Columns {
					t.Logf("  column: %s", c.Name)
				}
			}
		}
	}
}

func TestSqliteRealMigrations(t *testing.T) {
	migDir := filepath.Join("..", "..", "testdata", "sql", "migrations", "sqlite")
	if _, err := os.Stat(migDir); os.IsNotExist(err) {
		t.Skip("testdata sqlite migrations not found")
	}

	files, err := ResolveSchemaFiles(migDir)
	if err != nil {
		t.Fatalf("ResolveSchemaFiles error: %v", err)
	}

	// Filter only .up.sql
	var upFiles []string
	for _, f := range files {
		if strings.HasSuffix(f, ".up.sql") {
			upFiles = append(upFiles, f)
		}
	}
	if len(upFiles) == 0 {
		t.Skip("no SQLite .up.sql migration files found")
	}

	p := newSqliteParser()
	cat, err := p.ParseSchema(upFiles)
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	tableMap := make(map[string][]string)
	for _, table := range cat.Schemas[0].Tables {
		var cols []string
		for _, c := range table.Columns {
			cols = append(cols, c.Name)
		}
		tableMap[table.Name] = cols
	}

	// Verify all expected tables exist
	expectedTables := []string{"inbox", "todos", "notes", "workspaces", "tags", "todo_tags", "note_tags"}
	for _, name := range expectedTables {
		if _, ok := tableMap[name]; !ok {
			t.Errorf("missing table %q", name)
		}
	}

	// todos: 9 original + workspace_id (from 3) + priority (from 4) + recurrence_rule + recurrence_parent_id (from 6) = 13
	todoCols := tableMap["todos"]
	expectedTodoCols := []string{
		"id", "user_id", "title", "description", "status", "planned_date", "completed_at",
		"created_at", "updated_at",
		"workspace_id", "priority", "recurrence_rule", "recurrence_parent_id",
	}
	if len(todoCols) != len(expectedTodoCols) {
		t.Errorf("todos: expected %d columns, got %d: %v", len(expectedTodoCols), len(todoCols), todoCols)
	}
	for _, exp := range expectedTodoCols {
		found := false
		for _, c := range todoCols {
			if c == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("todos: missing column %q. Got: %v", exp, todoCols)
		}
	}

	// notes: 6 original + workspace_id (from 3) = 7
	noteCols := tableMap["notes"]
	expectedNoteCols := []string{"id", "user_id", "title", "body", "created_at", "updated_at", "workspace_id"}
	if len(noteCols) != len(expectedNoteCols) {
		t.Errorf("notes: expected %d columns, got %d: %v", len(expectedNoteCols), len(noteCols), noteCols)
	}
	for _, exp := range expectedNoteCols {
		found := slices.Contains(noteCols, exp)
		if !found {
			t.Errorf("notes: missing column %q. Got: %v", exp, noteCols)
		}
	}

	// workspaces: 7 columns
	if len(tableMap["workspaces"]) != 7 {
		t.Errorf("workspaces: expected 7 columns, got %d: %v", len(tableMap["workspaces"]), tableMap["workspaces"])
	}

	// inbox: 6 columns
	if len(tableMap["inbox"]) != 6 {
		t.Errorf("inbox: expected 6 columns, got %d: %v", len(tableMap["inbox"]), tableMap["inbox"])
	}

	// tags: 6 columns
	if len(tableMap["tags"]) != 6 {
		t.Errorf("tags: expected 6 columns, got %d: %v", len(tableMap["tags"]), tableMap["tags"])
	}
}

func TestSqliteTestdataFile(t *testing.T) {
	path := filepath.Join("testdata", "sqlite.sql")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("testdata file not found")
	}

	p := newSqliteParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	// Should have: users, posts, quoted_table (temp_data dropped, dup users ignored)
	tables := cat.Schemas[0].Tables
	if len(tables) != 3 {
		t.Errorf("expected 3 tables, got %d", len(tables))
		for _, tbl := range tables {
			t.Logf("  table: %s", tbl.Name)
		}
	}
}
