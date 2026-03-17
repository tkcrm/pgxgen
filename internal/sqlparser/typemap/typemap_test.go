package typemap

import (
	"testing"
)

func TestNewTypeMapper(t *testing.T) {
	tests := []struct {
		engine  string
		wantErr bool
	}{
		{"postgresql", false},
		{"", false},
		{"mysql", false},
		{"sqlite", false},
		{"oracle", true},
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			m, err := NewTypeMapper(tt.engine)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for engine %q", tt.engine)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if m == nil {
				t.Error("mapper should not be nil")
			}
		})
	}
}

func TestStructName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"user_status", "UserStatus"},
		{"id", "Id"},
		{"created_at", "CreatedAt"},
		{"book_type", "BookType"},
		{"a_b_c", "ABC"},
	}

	for _, tt := range tests {
		got := structName(tt.input)
		if got != tt.want {
			t.Errorf("structName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseDriver(t *testing.T) {
	tests := []struct {
		pkg  string
		want sqlDriver
	}{
		{"pgx/v4", driverPGXV4},
		{"pgx/v5", driverPGXV5},
		{"database/sql", driverDatabaseSQL},
		{"", driverDatabaseSQL},
		{"unknown", driverDatabaseSQL},
	}

	for _, tt := range tests {
		got := parseDriver(tt.pkg)
		if got != tt.want {
			t.Errorf("parseDriver(%q) = %d, want %d", tt.pkg, got, tt.want)
		}
	}
}
