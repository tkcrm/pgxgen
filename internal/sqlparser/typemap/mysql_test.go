package typemap

import (
	"testing"

	"github.com/tkcrm/pgxgen/internal/sqlparser/catalog"
)

func intPtr(v int) *int { return &v }

func TestMysqlIntegerTypes(t *testing.T) {
	m := &mysqlMapper{}
	opts := Options{}

	tests := []struct {
		typ      string
		notNull  bool
		unsigned bool
		want     string
	}{
		{"tinyint", true, false, "int8"},
		{"tinyint", true, true, "uint8"},
		{"tinyint", false, false, "sql.NullInt16"},
		{"smallint", true, false, "int16"},
		{"smallint", true, true, "uint16"},
		{"smallint", false, false, "sql.NullInt16"},
		{"int", true, false, "int32"},
		{"integer", true, false, "int32"},
		{"mediumint", true, false, "int32"},
		{"int", true, true, "uint32"},
		{"int", false, false, "sql.NullInt32"},
		{"bigint", true, false, "int64"},
		{"bigint", true, true, "uint64"},
		{"bigint", false, false, "sql.NullInt64"},
	}

	for _, tt := range tests {
		c := &catalog.Column{Type: tt.typ, NotNull: tt.notNull, IsUnsigned: tt.unsigned}
		got := m.GoType(c, noEnums, opts)
		if got != tt.want {
			t.Errorf("GoType(%q, notNull=%v, unsigned=%v) = %q, want %q",
				tt.typ, tt.notNull, tt.unsigned, got, tt.want)
		}
	}
}

func TestMysqlTinyintBool(t *testing.T) {
	m := &mysqlMapper{}
	opts := Options{}

	// tinyint(1) → bool
	c := &catalog.Column{Type: "tinyint", NotNull: true, Length: intPtr(1)}
	got := m.GoType(c, noEnums, opts)
	if got != "bool" {
		t.Errorf("tinyint(1) NOT NULL = %q, want bool", got)
	}

	c = &catalog.Column{Type: "tinyint", NotNull: false, Length: intPtr(1)}
	got = m.GoType(c, noEnums, opts)
	if got != "sql.NullBool" {
		t.Errorf("tinyint(1) NULL = %q, want sql.NullBool", got)
	}

	// tinyint (no length) → int8
	c = &catalog.Column{Type: "tinyint", NotNull: true}
	got = m.GoType(c, noEnums, opts)
	if got != "int8" {
		t.Errorf("tinyint NOT NULL = %q, want int8", got)
	}
}

func TestMysqlStringTypes(t *testing.T) {
	m := &mysqlMapper{}
	opts := Options{}

	for _, typ := range []string{"varchar", "text", "char", "tinytext", "mediumtext", "longtext"} {
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

func TestMysqlDateTimeTypes(t *testing.T) {
	m := &mysqlMapper{}
	opts := Options{}

	for _, typ := range []string{"date", "timestamp", "datetime", "time"} {
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

func TestMysqlDecimalType(t *testing.T) {
	m := &mysqlMapper{}
	opts := Options{}

	for _, typ := range []string{"decimal", "dec", "fixed"} {
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

func TestMysqlFloatTypes(t *testing.T) {
	m := &mysqlMapper{}
	opts := Options{}

	for _, typ := range []string{"double", "double precision", "real", "float"} {
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

func TestMysqlJsonType(t *testing.T) {
	m := &mysqlMapper{}
	got := m.GoType(col("json", true), noEnums, Options{})
	if got != "json.RawMessage" {
		t.Errorf("json = %q, want json.RawMessage", got)
	}
}

func TestMysqlBlobTypes(t *testing.T) {
	m := &mysqlMapper{}
	opts := Options{}

	for _, typ := range []string{"blob", "binary", "varbinary", "tinyblob", "mediumblob", "longblob"} {
		got := m.GoType(col(typ, true), noEnums, opts)
		if got != "[]byte" {
			t.Errorf("GoType(%q, NOT NULL) = %q, want []byte", typ, got)
		}

		got = m.GoType(col(typ, false), noEnums, opts)
		if got != "sql.NullString" {
			t.Errorf("GoType(%q, NULL) = %q, want sql.NullString", typ, got)
		}
	}
}

func TestMysqlBoolType(t *testing.T) {
	m := &mysqlMapper{}
	opts := Options{}

	got := m.GoType(col("boolean", true), noEnums, opts)
	if got != "bool" {
		t.Errorf("boolean NOT NULL = %q, want bool", got)
	}

	got = m.GoType(col("bool", false), noEnums, opts)
	if got != "sql.NullBool" {
		t.Errorf("bool NULL = %q, want sql.NullBool", got)
	}
}

func TestMysqlEnumColumn(t *testing.T) {
	m := &mysqlMapper{}
	got := m.GoType(col("enum", true), noEnums, Options{})
	if got != "string" {
		t.Errorf("enum = %q, want string", got)
	}
}

func TestMysqlUnknownType(t *testing.T) {
	m := &mysqlMapper{}
	got := m.GoType(col("unknown_type", true), noEnums, Options{})
	if got != "interface{}" {
		t.Errorf("unknown = %q, want interface{}", got)
	}
}
