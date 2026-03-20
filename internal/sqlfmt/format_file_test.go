package sqlfmt

import (
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
