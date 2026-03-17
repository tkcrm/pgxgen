package typemap

import (
	"testing"
)

func TestSqliteIntegerTypes(t *testing.T) {
	m := &sqliteMapper{}
	opts := Options{}

	for _, typ := range []string{"int", "integer", "tinyint", "smallint", "mediumint", "bigint", "int2", "int8"} {
		got := m.GoType(col(typ, true), noEnums, opts)
		if got != "int64" {
			t.Errorf("GoType(%q, NOT NULL) = %q, want int64", typ, got)
		}

		got = m.GoType(col(typ, false), noEnums, opts)
		if got != "sql.NullInt64" {
			t.Errorf("GoType(%q, NULL) = %q, want sql.NullInt64", typ, got)
		}
	}
}

func TestSqliteFloatTypes(t *testing.T) {
	m := &sqliteMapper{}
	opts := Options{}

	for _, typ := range []string{"real", "double", "doubleprecision", "float"} {
		got := m.GoType(col(typ, true), noEnums, opts)
		if got != "float64" {
			t.Errorf("GoType(%q, NOT NULL) = %q, want float64", typ, got)
		}

		got = m.GoType(col(typ, false), noEnums, opts)
		if got != "sql.NullFloat64" {
			t.Errorf("GoType(%q, NULL) = %q, want sql.NullFloat64", typ, got)
		}
	}
}

func TestSqliteStringTypes(t *testing.T) {
	m := &sqliteMapper{}
	opts := Options{}

	for _, typ := range []string{"text", "clob", "varchar(255)", "character(100)", "nvarchar(50)"} {
		got := m.GoType(col(typ, true), noEnums, opts)
		if got != "string" {
			t.Errorf("GoType(%q, NOT NULL) = %q, want string", typ, got)
		}

		got = m.GoType(col(typ, false), noEnums, opts)
		if got != "sql.NullString" {
			t.Errorf("GoType(%q, NULL) = %q, want sql.NullString", typ, got)
		}
	}
}

func TestSqliteBoolType(t *testing.T) {
	m := &sqliteMapper{}
	opts := Options{}

	got := m.GoType(col("boolean", true), noEnums, opts)
	if got != "bool" {
		t.Errorf("boolean NOT NULL = %q, want bool", got)
	}

	got = m.GoType(col("bool", true), noEnums, opts)
	if got != "bool" {
		t.Errorf("bool NOT NULL = %q, want bool", got)
	}

	got = m.GoType(col("boolean", false), noEnums, opts)
	if got != "sql.NullBool" {
		t.Errorf("boolean NULL = %q, want sql.NullBool", got)
	}
}

func TestSqliteDateTimeTypes(t *testing.T) {
	m := &sqliteMapper{}
	opts := Options{}

	for _, typ := range []string{"date", "datetime", "timestamp"} {
		got := m.GoType(col(typ, true), noEnums, opts)
		if got != "time.Time" {
			t.Errorf("GoType(%q, NOT NULL) = %q, want time.Time", typ, got)
		}

		got = m.GoType(col(typ, false), noEnums, opts)
		if got != "sql.NullTime" {
			t.Errorf("GoType(%q, NULL) = %q, want sql.NullTime", typ, got)
		}
	}
}

func TestSqliteJsonType(t *testing.T) {
	m := &sqliteMapper{}

	got := m.GoType(col("json", true), noEnums, Options{})
	if got != "json.RawMessage" {
		t.Errorf("json = %q, want json.RawMessage", got)
	}

	got = m.GoType(col("jsonb", true), noEnums, Options{})
	if got != "json.RawMessage" {
		t.Errorf("jsonb = %q, want json.RawMessage", got)
	}
}

func TestSqliteBlobType(t *testing.T) {
	m := &sqliteMapper{}
	got := m.GoType(col("blob", true), noEnums, Options{})
	if got != "[]byte" {
		t.Errorf("blob = %q, want []byte", got)
	}
}

func TestSqliteNullablePointers(t *testing.T) {
	m := &sqliteMapper{}
	opts := Options{EmitPointersForNull: true}

	tests := []struct {
		typ  string
		want string
	}{
		{"integer", "*int64"},
		{"real", "*float64"},
		{"text", "*string"},
		{"boolean", "*bool"},
		{"datetime", "*time.Time"},
	}

	for _, tt := range tests {
		got := m.GoType(col(tt.typ, false), noEnums, opts)
		if got != tt.want {
			t.Errorf("GoType(%q, NULL, pointers) = %q, want %q", tt.typ, got, tt.want)
		}
	}
}

func TestSqliteDecimalType(t *testing.T) {
	m := &sqliteMapper{}
	opts := Options{}

	got := m.GoType(col("decimal(10,2)", true), noEnums, opts)
	if got != "float64" {
		t.Errorf("decimal NOT NULL = %q, want float64", got)
	}

	got = m.GoType(col("numeric", true), noEnums, opts)
	if got != "float64" {
		t.Errorf("numeric NOT NULL = %q, want float64", got)
	}
}

func TestSqliteUnknownType(t *testing.T) {
	m := &sqliteMapper{}
	got := m.GoType(col("unknown_custom_type", true), noEnums, Options{})
	if got != "interface{}" {
		t.Errorf("unknown = %q, want interface{}", got)
	}
}

func TestSqliteAnyType(t *testing.T) {
	m := &sqliteMapper{}
	got := m.GoType(col("any", true), noEnums, Options{})
	if got != "interface{}" {
		t.Errorf("any = %q, want interface{}", got)
	}
}
