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
	if err := os.WriteFile(filepath.Join(dir, "001.sql"), []byte(`
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
	`), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	// Migration 2: add columns via ALTER TABLE
	if err := os.WriteFile(filepath.Join(dir, "002.sql"), []byte(`
		ALTER TABLE todos ADD COLUMN workspace_id TEXT;
		ALTER TABLE todos ADD COLUMN priority TEXT NOT NULL DEFAULT 'none';
		ALTER TABLE notes ADD COLUMN workspace_id TEXT;
	`), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

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

func TestSqliteColumnDefaults(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE items (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL DEFAULT 'untitled',
			count INTEGER NOT NULL DEFAULT 0,
			price REAL NOT NULL DEFAULT 9.99,
			active BOOLEAN NOT NULL DEFAULT FALSE,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)

	p := newSqliteParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	table := cat.Schemas[0].Tables[0]
	defaults := map[string]string{
		"name":       "'untitled'",
		"count":      "0",
		"price":      "9.99",
		"active":     "FALSE",
		"created_at": "CURRENT_TIMESTAMP",
	}
	for _, col := range table.Columns {
		if expected, ok := defaults[col.Name]; ok {
			if !strings.EqualFold(col.Default, expected) {
				t.Errorf("column %q: expected default %q, got %q", col.Name, expected, col.Default)
			}
		}
	}
}

func TestSqlitePrimaryKeyInline(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		);
	`)

	p := newSqliteParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	table := cat.Schemas[0].Tables[0]
	if table.PrimaryKey == nil {
		t.Fatal("expected PrimaryKey to be set")
	}
	if len(table.PrimaryKey.Columns) != 1 || table.PrimaryKey.Columns[0] != "id" {
		t.Errorf("expected PK [id], got %v", table.PrimaryKey.Columns)
	}
	if !table.Columns[0].IsPrimary {
		t.Error("id should have IsPrimary=true")
	}
}

func TestSqlitePrimaryKeyComposite(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE order_items (
			order_id INTEGER NOT NULL,
			product_id INTEGER NOT NULL,
			quantity INTEGER NOT NULL,
			PRIMARY KEY (order_id, product_id)
		);
	`)

	p := newSqliteParser()
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
}

func TestSqliteForeignKeyInline(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE parents (id INTEGER PRIMARY KEY);
		CREATE TABLE children (
			id INTEGER PRIMARY KEY,
			parent_id INTEGER NOT NULL REFERENCES parents(id) ON DELETE CASCADE
		);
	`)

	p := newSqliteParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	children := cat.Schemas[0].Tables[1]
	if len(children.ForeignKeys) != 1 {
		t.Fatalf("expected 1 FK, got %d", len(children.ForeignKeys))
	}
	fk := children.ForeignKeys[0]
	if fk.RefTable != "parents" {
		t.Errorf("expected ref table 'parents', got %q", fk.RefTable)
	}
	if len(fk.Columns) != 1 || fk.Columns[0] != "parent_id" {
		t.Errorf("expected FK columns [parent_id], got %v", fk.Columns)
	}
	if fk.OnDelete != "CASCADE" {
		t.Errorf("expected ON DELETE CASCADE, got %q", fk.OnDelete)
	}
}

func TestSqliteForeignKeyTableLevel(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE parents (id INTEGER PRIMARY KEY);
		CREATE TABLE children (
			id INTEGER PRIMARY KEY,
			parent_id INTEGER NOT NULL,
			CONSTRAINT fk_parent FOREIGN KEY (parent_id) REFERENCES parents(id) ON DELETE SET NULL
		);
	`)

	p := newSqliteParser()
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
	if fk.OnDelete != "SET NULL" {
		t.Errorf("expected ON DELETE SET NULL, got %q", fk.OnDelete)
	}
}

func TestSqliteUniqueConstraint(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			username TEXT NOT NULL,
			UNIQUE (username)
		);
	`)

	p := newSqliteParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	table := cat.Schemas[0].Tables[0]
	if len(table.Uniques) != 2 {
		t.Fatalf("expected 2 unique constraints, got %d", len(table.Uniques))
	}
}

func TestSqliteCheckConstraint(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE products (
			id INTEGER PRIMARY KEY,
			price REAL NOT NULL,
			CONSTRAINT positive_price CHECK (price > 0)
		);
	`)

	p := newSqliteParser()
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
}

func TestSqliteCreateIndex(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			email TEXT NOT NULL,
			created_at DATETIME NOT NULL
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_email ON users(email);
		CREATE INDEX idx_created ON users(created_at);
	`)

	p := newSqliteParser()
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

	idx1 := table.Indexes[1]
	if idx1.IsUnique {
		t.Error("idx_created should not be unique")
	}
}

func TestSqliteDropIndex(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT NOT NULL);
		CREATE INDEX idx_email ON users(email);
		CREATE INDEX idx_keep ON users(id);
		DROP INDEX idx_email;
	`)

	p := newSqliteParser()
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

func TestSqliteAlterTableAddColumnWithFK(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE workspaces (id TEXT PRIMARY KEY);
		CREATE TABLE todos (id TEXT PRIMARY KEY, title TEXT NOT NULL);
		ALTER TABLE todos ADD COLUMN workspace_id TEXT REFERENCES workspaces(id) ON DELETE SET NULL;
	`)

	p := newSqliteParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	todos := cat.Schemas[0].Tables[1]
	if len(todos.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(todos.Columns))
	}
	if len(todos.ForeignKeys) != 1 {
		t.Fatalf("expected 1 FK, got %d", len(todos.ForeignKeys))
	}
	fk := todos.ForeignKeys[0]
	if fk.RefTable != "workspaces" {
		t.Errorf("expected ref table 'workspaces', got %q", fk.RefTable)
	}
	if fk.OnDelete != "SET NULL" {
		t.Errorf("expected ON DELETE SET NULL, got %q", fk.OnDelete)
	}
}

func TestSqliteFullMigrationsExtended(t *testing.T) {
	migrationDir := filepath.Join("..", "..", "testdata", "sql", "migrations", "sqlite")
	if _, err := os.Stat(migrationDir); os.IsNotExist(err) {
		t.Skip("migration directory not found, skipping")
	}

	files, err := ResolveSchemaFiles(migrationDir)
	if err != nil {
		t.Fatalf("ResolveSchemaFiles error: %v", err)
	}

	p := newSqliteParser()
	cat, err := p.ParseSchema(files)
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	schema := cat.Schemas[0]

	// Check all tables exist
	tableNames := make(map[string]bool)
	for _, t := range schema.Tables {
		tableNames[t.Name] = true
	}
	expected := []string{"inbox", "todos", "notes", "workspaces", "tags", "todo_tags", "note_tags"}
	for _, name := range expected {
		if !tableNames[name] {
			t.Errorf("table %q not found", name)
		}
	}

	// Check todos has indexes
	for _, tbl := range schema.Tables {
		if tbl.Name == "todos" {
			if len(tbl.Indexes) == 0 {
				t.Error("todos should have indexes")
			}
			// Should have FK to workspaces (from ALTER TABLE ADD COLUMN)
			if len(tbl.ForeignKeys) == 0 {
				t.Error("todos should have foreign keys")
			}
			// Should have priority column with default (from ALTER TABLE ADD COLUMN)
			found := false
			for _, col := range tbl.Columns {
				if col.Name == "priority" && col.Default != "" {
					found = true
				}
			}
			if !found {
				t.Error("todos should have priority column with default")
			}
		}
		if tbl.Name == "todo_tags" {
			// Should have composite PK
			if tbl.PrimaryKey == nil || len(tbl.PrimaryKey.Columns) != 2 {
				t.Error("todo_tags should have composite PK")
			}
			// Should have 2 FKs with ON DELETE CASCADE
			if len(tbl.ForeignKeys) != 2 {
				t.Errorf("todo_tags: expected 2 FKs, got %d", len(tbl.ForeignKeys))
			}
		}
	}
}
