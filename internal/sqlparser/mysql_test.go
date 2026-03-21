package sqlparser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMysqlCreateTable(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE users (
			id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			username VARCHAR(255) NOT NULL,
			email VARCHAR(255) NULL,
			age INT NOT NULL DEFAULT 0,
			score DOUBLE NOT NULL DEFAULT 0.0,
			balance DECIMAL(10,2) NOT NULL,
			bio TEXT NULL,
			avatar BLOB NULL,
			metadata JSON NULL,
			is_active BOOLEAN NOT NULL DEFAULT FALSE,
			small_val SMALLINT UNSIGNED NOT NULL,
			big_val BIGINT UNSIGNED NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NULL
		);
	`)

	p := newMysqlParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	tables := cat.Schemas[0].Tables
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}

	table := tables[0]
	if table.Name != "users" {
		t.Errorf("expected table 'users', got %q", table.Name)
	}

	if len(table.Columns) != 14 {
		t.Fatalf("expected 14 columns, got %d", len(table.Columns))
	}

	// Check specific columns
	id := table.Columns[0]
	if id.Name != "id" || !id.NotNull {
		t.Errorf("id column: expected NOT NULL, got %+v", id)
	}

	username := table.Columns[1]
	if username.Name != "username" || username.Type != "varchar" || !username.NotNull {
		t.Errorf("username: expected varchar NOT NULL, got type=%q notNull=%v", username.Type, username.NotNull)
	}

	email := table.Columns[2]
	if email.NotNull {
		t.Error("email should be nullable")
	}

	// Unsigned columns
	smallVal := table.Columns[10]
	if !smallVal.IsUnsigned {
		t.Error("small_val should be unsigned")
	}

	bigVal := table.Columns[11]
	if !bigVal.IsUnsigned {
		t.Error("big_val should be unsigned")
	}
}

func TestMysqlTinyintBool(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE flags (
			id BIGINT NOT NULL PRIMARY KEY,
			is_active TINYINT(1) NOT NULL DEFAULT 0,
			role_level TINYINT NOT NULL DEFAULT 0
		);
	`)

	p := newMysqlParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	table := cat.Schemas[0].Tables[0]

	isActive := table.Columns[1]
	if isActive.Length == nil || *isActive.Length != 1 {
		t.Error("is_active TINYINT(1) should have Length=1")
	}

	roleLevel := table.Columns[2]
	if roleLevel.Length != nil && *roleLevel.Length == 1 {
		t.Error("role_level TINYINT should not have Length=1")
	}
}

func TestMysqlTableComment(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE users (
			id BIGINT NOT NULL PRIMARY KEY
		) COMMENT='Main users table';
	`)

	p := newMysqlParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	table := cat.Schemas[0].Tables[0]
	if table.Comment != "Main users table" {
		t.Errorf("expected table comment 'Main users table', got %q", table.Comment)
	}
}

func TestMysqlAlterTable(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE profiles (
			id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			user_id BIGINT NOT NULL
		);
		ALTER TABLE profiles ADD COLUMN bio TEXT NULL;
		ALTER TABLE profiles ADD COLUMN website VARCHAR(255) NOT NULL;
		ALTER TABLE profiles DROP COLUMN website;
		ALTER TABLE profiles MODIFY COLUMN bio MEDIUMTEXT NULL;
	`)

	p := newMysqlParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	table := cat.Schemas[0].Tables[0]

	// After operations: id, user_id, bio (website added then dropped)
	if len(table.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(table.Columns))
	}

	bio := table.Columns[2]
	if bio.Name != "bio" {
		t.Errorf("expected column 'bio', got %q", bio.Name)
	}
	// After MODIFY COLUMN, type should be mediumtext
	if bio.Type != "mediumtext" {
		t.Errorf("expected bio type 'mediumtext' after modify, got %q", bio.Type)
	}
}

func TestMysqlDropTable(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE temp_data (id BIGINT NOT NULL PRIMARY KEY, data TEXT);
		CREATE TABLE keep_me (id BIGINT NOT NULL PRIMARY KEY);
		DROP TABLE temp_data;
	`)

	p := newMysqlParser()
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

func TestMysqlIfNotExists(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE users (id BIGINT NOT NULL PRIMARY KEY, name VARCHAR(255) NOT NULL);
		CREATE TABLE IF NOT EXISTS users (id BIGINT NOT NULL PRIMARY KEY, different TEXT);
	`)

	p := newMysqlParser()
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
}

func TestMysqlCreateView(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE users (
			id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			username VARCHAR(255) NOT NULL,
			email VARCHAR(255) NULL
		);

		CREATE VIEW active_users AS
			SELECT id, username, email FROM users;
	`)

	p := newMysqlParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	views := cat.Schemas[0].Views
	if len(views) != 1 {
		t.Fatalf("expected 1 view, got %d", len(views))
	}

	view := views[0]
	if view.Name != "active_users" {
		t.Errorf("expected view name 'active_users', got %q", view.Name)
	}
	if len(view.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(view.Columns))
	}
	if view.Columns[0].Name != "id" {
		t.Errorf("expected first column 'id', got %q", view.Columns[0].Name)
	}
	if view.Query == "" {
		t.Error("expected non-empty query")
	}
}

func TestMysqlDropView(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE t1 (id BIGINT NOT NULL PRIMARY KEY);
		CREATE VIEW v1 AS SELECT id FROM t1;
		DROP VIEW v1;
	`)

	p := newMysqlParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	if len(cat.Schemas[0].Views) != 0 {
		t.Errorf("expected 0 views after DROP VIEW, got %d", len(cat.Schemas[0].Views))
	}
}

func TestMysqlCreateViewStar(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE orders (
			id BIGINT NOT NULL,
			amount DECIMAL(10,2) NOT NULL,
			status VARCHAR(50) NOT NULL
		);
		CREATE VIEW all_orders AS SELECT * FROM orders;
	`)

	p := newMysqlParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	views := cat.Schemas[0].Views
	if len(views) != 1 {
		t.Fatalf("expected 1 view, got %d", len(views))
	}
	if len(views[0].Columns) != 3 {
		t.Fatalf("expected 3 columns from SELECT *, got %d", len(views[0].Columns))
	}
}

func TestMysqlCreateViewWithJoin(t *testing.T) {
	path := writeTemp(t, `
		CREATE TABLE webhooks (
			id BIGINT NOT NULL PRIMARY KEY,
			kind VARCHAR(50) NOT NULL,
			client_id BIGINT NOT NULL
		);
		CREATE TABLE clients (
			id BIGINT NOT NULL PRIMARY KEY,
			callback_url TEXT NOT NULL,
			secret_key TEXT NOT NULL
		);
		CREATE VIEW webhook_view AS
			SELECT w.*, c.callback_url, c.secret_key
			FROM webhooks w
			JOIN clients c ON c.id = w.client_id;
	`)

	p := newMysqlParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	views := cat.Schemas[0].Views
	if len(views) != 1 {
		t.Fatalf("expected 1 view, got %d", len(views))
	}
	// w.* = 3 columns + 2 explicit = 5
	if len(views[0].Columns) != 5 {
		t.Fatalf("expected 5 columns, got %d", len(views[0].Columns))
	}
	if views[0].Columns[0].Name != "id" {
		t.Errorf("expected first column 'id', got %q", views[0].Columns[0].Name)
	}
	if views[0].Columns[3].Name != "callback_url" {
		t.Errorf("expected column 'callback_url', got %q", views[0].Columns[3].Name)
	}
}

func TestMysqlTestdataFile(t *testing.T) {
	path := filepath.Join("testdata", "mysql.sql")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("testdata file not found")
	}

	p := newMysqlParser()
	cat, err := p.ParseSchema([]string{path})
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	// Should have: users, posts, profiles (temp_data dropped, duplicate users ignored)
	tables := cat.Schemas[0].Tables
	if len(tables) != 3 {
		t.Errorf("expected 3 tables, got %d", len(tables))
		for _, tbl := range tables {
			t.Logf("  table: %s", tbl.Name)
		}
	}
}
