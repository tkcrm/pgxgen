package sqlfmt

import (
	"strings"
	"testing"

	"github.com/tkcrm/pgxgen/internal/sqlfmt/formatters"
)

func TestFormatSingleStatementWithDashComment(t *testing.T) {
	sql := "-- Create users table\ncreate table users(id integer, name varchar(255))"
	opts := formatters.DefaultOptions()
	result, err := Format(sql, opts)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	want := "-- Create users table\nCREATE TABLE users (\n  id INTEGER,\n  name VARCHAR(255)\n)"
	if result != want {
		t.Errorf("\n=== GOT ===\n%s\n=== WANT ===\n%s", result, want)
	}
}

func TestFormatFile(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr bool
	}{
		{
			name: "single statement",
			src:  "select * from users;\n",
		},
		{
			name: "multiple statements",
			src:  "create table users(id integer, name text);\n\nselect * from users;\n",
		},
		{
			name: "comment between statements",
			src:  "create table users(id integer);\n\n-- query\nselect * from users;\n",
		},
		{
			name: "comment before first statement",
			src:  "-- schema\ncreate table users(id integer);\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := formatters.DefaultOptions()
			got, err := FormatFile([]byte(tt.src), opts)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("FormatFile error: %v", err)
			}
			t.Logf("Result:\n%s", string(got))
		})
	}
}

func TestFormatFileIdempotent(t *testing.T) {
	src := "create table users(id integer, name text);\n\nselect * from users where id > 0;\n"
	opts := formatters.DefaultOptions()

	first, err := FormatFile([]byte(src), opts)
	if err != nil {
		t.Fatalf("first format error: %v", err)
	}

	second, err := FormatFile(first, opts)
	if err != nil {
		t.Fatalf("second format error: %v", err)
	}

	if string(first) != string(second) {
		t.Errorf("not idempotent:\n=== FIRST ===\n%s\n=== SECOND ===\n%s", string(first), string(second))
	}
}

func TestFormatFileCheck(t *testing.T) {
	// Already formatted
	formatted := "SELECT\n  *\nFROM users;\n"
	opts := formatters.DefaultOptions()

	result, err := FormatFile([]byte(formatted), opts)
	if err != nil {
		t.Fatalf("FormatFile error: %v", err)
	}

	if string(result) != formatted {
		t.Errorf("already formatted content changed:\n=== GOT ===\n%s\n=== WANT ===\n%s", string(result), formatted)
	}
}

func TestFormatFileBlockComment(t *testing.T) {
	src := "/*\n * Multi-line comment\n * spanning several lines\n */\nselect * from users;\n"
	opts := formatters.DefaultOptions()
	got, err := FormatFile([]byte(src), opts)
	if err != nil {
		t.Fatalf("FormatFile error: %v", err)
	}
	t.Logf("Result:\n%s", string(got))
	// Verify it doesn't error and produces output
	if len(got) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestFormatFileNoTrailingNewline(t *testing.T) {
	src := "select * from users"
	opts := formatters.DefaultOptions()
	got, err := FormatFile([]byte(src), opts)
	if err != nil {
		t.Fatalf("FormatFile error: %v", err)
	}
	// Should still produce valid output with trailing newline
	result := string(got)
	if result[len(result)-1] != '\n' {
		t.Error("expected trailing newline in output")
	}
}

// Bug fix: sqlc annotations (-- name: XXX :exec) with $1/$2 params must preserve semantic equality
func TestFormatSqlcAnnotatedQuery(t *testing.T) {
	sql := "-- name: SetStatus :exec\nupdate transfers set updated_at = now(), status = $2 where id = $1"
	opts := formatters.DefaultOptions()
	result, err := Format(sql, opts)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	if !strings.Contains(result, "$1") || !strings.Contains(result, "$2") {
		t.Errorf("lost dollar params in result:\n%s", result)
	}
	if !strings.Contains(result, "-- name: SetStatus :exec") {
		t.Errorf("lost sqlc annotation in result:\n%s", result)
	}
}

// Bug fix: TIMESTAMP WITH TIME ZONE must not be broken across lines
func TestFormatTimestampWithTimeZone(t *testing.T) {
	sql := "create table t(created_at timestamp with time zone not null default now())"
	opts := formatters.DefaultOptions()
	result, err := Format(sql, opts)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	if !strings.Contains(result, "TIMESTAMP WITH TIME ZONE") {
		t.Errorf("TIMESTAMP WITH TIME ZONE broken:\n%s", result)
	}
}

// Bug fix: PL/pgSQL DO blocks must not cause errors (kept unchanged)
func TestFormatFileDoBlock(t *testing.T) {
	src := "DO $$ BEGIN RAISE NOTICE 'hello'; END $$;\n\nSELECT 1;\n"
	opts := formatters.DefaultOptions()
	got, err := FormatFile([]byte(src), opts)
	if err != nil {
		t.Fatalf("FormatFile error: %v", err)
	}
	// DO block should be preserved (not formatted, not errored)
	if !strings.Contains(string(got), "DO $$") {
		t.Errorf("DO block lost:\n%s", string(got))
	}
}

// Bug fix: JSON operators -> and ->> must be preserved as single tokens
func TestFormatJsonOperators(t *testing.T) {
	sql := "select data->'key'->>'value' from test"
	opts := formatters.DefaultOptions()
	result, err := Format(sql, opts)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	if !strings.Contains(result, "->") {
		t.Errorf("JSON arrow operator lost:\n%s", result)
	}
	if !strings.Contains(result, "->>") {
		t.Errorf("JSON double arrow operator lost:\n%s", result)
	}
	// Must not have "- >" (dash space gt)
	if strings.Contains(result, "- >") {
		t.Errorf("JSON arrow split into separate tokens:\n%s", result)
	}
}

// Bug fix: JSON operators idempotency — already formatted "- >" should become "->"
func TestFormatJsonOperatorsIdempotent(t *testing.T) {
	sql := "SELECT data -> 'key' ->> 'value' FROM test"
	opts := formatters.DefaultOptions()
	first, err := Format(sql, opts)
	if err != nil {
		t.Fatalf("first format error: %v", err)
	}
	second, err := Format(first, opts)
	if err != nil {
		t.Fatalf("second format error: %v", err)
	}
	if first != second {
		t.Errorf("not idempotent:\n=== FIRST ===\n%s\n=== SECOND ===\n%s", first, second)
	}
}

// Bug fix: minus before INTERVAL must not be treated as JSON arrow
func TestFormatMinusInterval(t *testing.T) {
	sql := "select now() - interval '30 days' from t"
	opts := formatters.DefaultOptions()
	result, err := Format(sql, opts)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	// Should have space around minus: "NOW() - INTERVAL"
	if strings.Contains(result, "-INTERVAL") || strings.Contains(result, "- INTERVAL") {
		// "- INTERVAL" with single space is acceptable, "-INTERVAL" is not
		if strings.Contains(result, "-INTERVAL") {
			t.Errorf("minus stuck to INTERVAL:\n%s", result)
		}
	}
}

// Bug fix: FormatFile with multiple sqlc queries preserves all annotations
func TestFormatFileSqlcMultipleQueries(t *testing.T) {
	src := `-- name: GetByID :one
select * from users where id = $1;

-- name: SetName :exec
update users set name = $2 where id = $1;
`
	opts := formatters.DefaultOptions()
	got, err := FormatFile([]byte(src), opts)
	if err != nil {
		t.Fatalf("FormatFile error: %v", err)
	}
	result := string(got)
	if !strings.Contains(result, "-- name: GetByID :one") {
		t.Error("lost first sqlc annotation")
	}
	if !strings.Contains(result, "-- name: SetName :exec") {
		t.Error("lost second sqlc annotation")
	}
	if !strings.Contains(result, "$1") {
		t.Error("lost $1 param")
	}
	if !strings.Contains(result, "$2") {
		t.Error("lost $2 param")
	}
}

// Bug fix: Generic formatter adds newline after standalone comment before CREATE/ALTER/DROP
func TestFormatCommentBeforeCreate(t *testing.T) {
	sql := "-- users table\ncreate table users(id integer)"
	opts := formatters.DefaultOptions()
	result, err := Format(sql, opts)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	// Comment and CREATE should be on separate lines
	lines := strings.Split(result, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected multiple lines, got:\n%s", result)
	}
	if !strings.HasPrefix(lines[0], "-- users table") {
		t.Errorf("first line should be comment, got: %s", lines[0])
	}
	if !strings.HasPrefix(lines[1], "CREATE") {
		t.Errorf("second line should start with CREATE, got: %s", lines[1])
	}
}

// Bug fix: SQL keywords must be uppercased (DEFAULT, CHECK, CONSTRAINT, etc.)
func TestFormatKeywordsUppercased(t *testing.T) {
	sql := "create table t(id integer not null default 0 check (id >= 0), name varchar(255) unique, ref_id integer references other_table, constraint pk primary key(id))"
	opts := formatters.DefaultOptions()
	result, err := Format(sql, opts)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	for _, kw := range []string{"DEFAULT", "CHECK", "UNIQUE", "REFERENCES", "CONSTRAINT"} {
		if !strings.Contains(result, kw) {
			t.Errorf("keyword %s not uppercased in:\n%s", kw, result)
		}
	}
}

// Bug fix: SQL types must be uppercased (UUID, JSONB, BOOL, BIGINT, etc.)
func TestFormatTypesUppercased(t *testing.T) {
	sql := "create table t(id uuid, data jsonb, active bool, count bigint)"
	opts := formatters.DefaultOptions()
	result, err := Format(sql, opts)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	for _, tp := range []string{"UUID", "JSONB", "BOOL", "BIGINT"} {
		if !strings.Contains(result, tp) {
			t.Errorf("type %s not uppercased in:\n%s", tp, result)
		}
	}
}

// Integration: full idempotency test with complex real-world SQL
func TestFormatFileIdempotentComplex(t *testing.T) {
	src := `-- Create users
create table users(
  id uuid not null primary key default gen_random_uuid(),
  name varchar(255) not null check (name != ''),
  data jsonb not null default '{}',
  created_at timestamp with time zone not null default now(),
  updated_at timestamp with time zone
);
create index users_name_idx on users using btree (name);

-- name: GetByID :one
select * from users where id = $1;

-- name: SetData :exec
update users set data = $2, updated_at = now() where id = $1;

-- name: GetJsonField :one
select data->'key'->>'value' from users where id = $1;

-- name: CleanOld :exec
delete from users where created_at < now() - interval '30 days';
`
	opts := formatters.DefaultOptions()

	first, err := FormatFile([]byte(src), opts)
	if err != nil {
		t.Fatalf("first format error: %v", err)
	}

	second, err := FormatFile(first, opts)
	if err != nil {
		t.Fatalf("second format error: %v", err)
	}

	if string(first) != string(second) {
		t.Errorf("not idempotent:\n=== FIRST ===\n%s\n=== SECOND ===\n%s", string(first), string(second))
	}

	result := string(first)
	// Verify key formatting rules
	if !strings.Contains(result, "TIMESTAMP WITH TIME ZONE") {
		t.Error("TIMESTAMP WITH TIME ZONE broken")
	}
	if !strings.Contains(result, "->") && !strings.Contains(result, "->>") {
		t.Error("JSON operators missing")
	}
	if strings.Contains(result, "- >") {
		t.Error("JSON operator split")
	}
	if !strings.Contains(result, "INTERVAL") {
		t.Error("INTERVAL missing")
	}
	if !strings.Contains(result, "$1") || !strings.Contains(result, "$2") {
		t.Error("dollar params lost")
	}
}

func TestIsCommentOnly(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"single line dash comment", "-- comment", true},
		{"single line slash comment", "// comment", true},
		{"block comment single line", "/* comment */", true},
		{"block comment multi line", "/*\n * line 1\n * line 2\n */", true},
		{"not a comment", "SELECT * FROM users", false},
		{"mixed comment and sql", "-- comment\nSELECT 1", false},
		{"empty string", "", true},
		{"only whitespace", "  \n  \n  ", true},
		{"block comment with content after", "/* comment */ SELECT 1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCommentOnly(tt.s)
			if got != tt.want {
				t.Errorf("isCommentOnly(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}
