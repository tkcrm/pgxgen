package sqlparser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewParser(t *testing.T) {
	tests := []struct {
		engine  string
		wantErr bool
	}{
		{"postgresql", false},
		{"", false}, // default is postgresql
		{"mysql", false},
		{"sqlite", false},
		{"oracle", true},
		{"mssql", true},
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			p, err := NewParser(tt.engine)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewParser(%q) expected error, got nil", tt.engine)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewParser(%q) unexpected error: %v", tt.engine, err)
			}
			if p == nil {
				t.Errorf("NewParser(%q) returned nil parser", tt.engine)
			}
		})
	}
}

func TestResolveSchemaFiles_Directory(t *testing.T) {
	dir := t.TempDir()

	// Create mixed files
	for _, name := range []string{"002_second.sql", "001_first.sql", "readme.txt", "data.csv", "003_third.SQL"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("-- test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := ResolveSchemaFiles(dir)
	if err != nil {
		t.Fatalf("ResolveSchemaFiles error: %v", err)
	}

	// .sql files sorted (note: .SQL uppercase also matches on case-insensitive fs)
	if len(files) < 2 {
		t.Fatalf("expected at least 2 .sql files, got %d: %v", len(files), files)
	}

	if filepath.Base(files[0]) != "001_first.sql" {
		t.Errorf("expected first file 001_first.sql, got %s", filepath.Base(files[0]))
	}
	if filepath.Base(files[1]) != "002_second.sql" {
		t.Errorf("expected second file 002_second.sql, got %s", filepath.Base(files[1]))
	}
}

func TestResolveSchemaFiles_SingleFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "schema.sql")
	if err := os.WriteFile(filePath, []byte("-- test"), 0644); err != nil {
		t.Fatal(err)
	}

	files, err := ResolveSchemaFiles(filePath)
	if err != nil {
		t.Fatalf("ResolveSchemaFiles error: %v", err)
	}

	if len(files) != 1 || files[0] != filePath {
		t.Errorf("expected [%s], got %v", filePath, files)
	}
}

func TestResolveSchemaFiles_NotExists(t *testing.T) {
	_, err := ResolveSchemaFiles("/nonexistent/path")
	if err == nil {
		t.Error("expected error for non-existent path")
	}
}

func TestResolveSchemaFiles_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	files, err := ResolveSchemaFiles(dir)
	if err != nil {
		t.Fatalf("ResolveSchemaFiles error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}
