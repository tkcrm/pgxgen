package typemap

import (
	"testing"

	"github.com/tkcrm/pgxgen/internal/sqlparser/catalog"
)

func col(typ string, notNull bool) *catalog.Column {
	return &catalog.Column{Type: typ, NotNull: notNull}
}

var noEnums []*catalog.Enum

func TestPostgresIntegerTypes(t *testing.T) {
	m := &postgresMapper{}
	defaultOpts := Options{}

	tests := []struct {
		typ     string
		notNull bool
		want    string
	}{
		{"serial", true, "int32"},
		{"serial", false, "sql.NullInt32"},
		{"serial4", true, "int32"},
		{"bigserial", true, "int64"},
		{"bigserial", false, "sql.NullInt64"},
		{"serial8", true, "int64"},
		{"smallserial", true, "int16"},
		{"serial2", true, "int16"},
		{"integer", true, "int32"},
		{"int", true, "int32"},
		{"int4", true, "int32"},
		{"integer", false, "sql.NullInt32"},
		{"bigint", true, "int64"},
		{"int8", true, "int64"},
		{"bigint", false, "sql.NullInt64"},
		{"smallint", true, "int16"},
		{"int2", true, "int16"},
		{"smallint", false, "sql.NullInt16"},
	}

	for _, tt := range tests {
		got := m.GoType(col(tt.typ, tt.notNull), noEnums, defaultOpts)
		if got != tt.want {
			t.Errorf("GoType(%q, notNull=%v) = %q, want %q", tt.typ, tt.notNull, got, tt.want)
		}
	}
}

func TestPostgresFloatTypes(t *testing.T) {
	m := &postgresMapper{}
	defaultOpts := Options{}

	tests := []struct {
		typ     string
		notNull bool
		want    string
	}{
		{"float8", true, "float64"},
		{"double precision", true, "float64"},
		{"float8", false, "sql.NullFloat64"},
		{"float4", true, "float32"},
		{"real", true, "float32"},
		{"float4", false, "sql.NullFloat64"},
		{"numeric", true, "string"},
		{"money", true, "string"},
		{"numeric", false, "sql.NullString"},
	}

	for _, tt := range tests {
		got := m.GoType(col(tt.typ, tt.notNull), noEnums, defaultOpts)
		if got != tt.want {
			t.Errorf("GoType(%q, notNull=%v) = %q, want %q", tt.typ, tt.notNull, got, tt.want)
		}
	}
}

func TestPostgresBoolType(t *testing.T) {
	m := &postgresMapper{}

	got := m.GoType(col("boolean", true), noEnums, Options{})
	if got != "bool" {
		t.Errorf("bool NOT NULL = %q, want bool", got)
	}

	got = m.GoType(col("bool", false), noEnums, Options{})
	if got != "sql.NullBool" {
		t.Errorf("bool NULL = %q, want sql.NullBool", got)
	}

	got = m.GoType(col("bool", false), noEnums, Options{SqlPackage: "pgx/v5"})
	if got != "pgtype.Bool" {
		t.Errorf("bool NULL pgxv5 = %q, want pgtype.Bool", got)
	}
}

func TestPostgresStringTypes(t *testing.T) {
	m := &postgresMapper{}
	defaultOpts := Options{}

	for _, typ := range []string{"text", "varchar", "character varying", "citext", "name"} {
		got := m.GoType(col(typ, true), noEnums, defaultOpts)
		if got != "string" {
			t.Errorf("GoType(%q, NOT NULL) = %q, want string", typ, got)
		}

		got = m.GoType(col(typ, false), noEnums, defaultOpts)
		if got != "sql.NullString" {
			t.Errorf("GoType(%q, NULL) = %q, want sql.NullString", typ, got)
		}
	}

	// pgx/v5 nullable
	got := m.GoType(col("text", false), noEnums, Options{SqlPackage: "pgx/v5"})
	if got != "pgtype.Text" {
		t.Errorf("text NULL pgxv5 = %q, want pgtype.Text", got)
	}
}

func TestPostgresJsonTypes(t *testing.T) {
	m := &postgresMapper{}

	// pgx/v5
	got := m.GoType(col("json", true), noEnums, Options{SqlPackage: "pgx/v5"})
	if got != "[]byte" {
		t.Errorf("json pgxv5 = %q, want []byte", got)
	}

	got = m.GoType(col("jsonb", true), noEnums, Options{SqlPackage: "pgx/v5"})
	if got != "[]byte" {
		t.Errorf("jsonb pgxv5 = %q, want []byte", got)
	}

	// pgx/v4
	got = m.GoType(col("json", true), noEnums, Options{SqlPackage: "pgx/v4"})
	if got != "pgtype.JSON" {
		t.Errorf("json pgxv4 = %q, want pgtype.JSON", got)
	}

	got = m.GoType(col("jsonb", true), noEnums, Options{SqlPackage: "pgx/v4"})
	if got != "pgtype.JSONB" {
		t.Errorf("jsonb pgxv4 = %q, want pgtype.JSONB", got)
	}

	// database/sql
	got = m.GoType(col("json", true), noEnums, Options{})
	if got != "json.RawMessage" {
		t.Errorf("json default = %q, want json.RawMessage", got)
	}
}

func TestPostgresDateTimeTypes(t *testing.T) {
	m := &postgresMapper{}

	tests := []struct {
		typ     string
		notNull bool
		pkg     string
		want    string
	}{
		{"date", true, "", "time.Time"},
		{"date", false, "", "sql.NullTime"},
		{"date", true, "pgx/v5", "pgtype.Date"},
		{"timestamp", true, "", "time.Time"},
		{"timestamp", false, "", "sql.NullTime"},
		{"timestamp", true, "pgx/v5", "pgtype.Timestamp"},
		{"timestamptz", true, "", "time.Time"},
		{"timestamptz", false, "", "sql.NullTime"},
		{"timestamptz", true, "pgx/v5", "pgtype.Timestamptz"},
	}

	for _, tt := range tests {
		got := m.GoType(col(tt.typ, tt.notNull), noEnums, Options{SqlPackage: tt.pkg})
		if got != tt.want {
			t.Errorf("GoType(%q, notNull=%v, pkg=%q) = %q, want %q", tt.typ, tt.notNull, tt.pkg, got, tt.want)
		}
	}
}

func TestPostgresUUID(t *testing.T) {
	m := &postgresMapper{}

	got := m.GoType(col("uuid", true), noEnums, Options{SqlPackage: "pgx/v5"})
	if got != "pgtype.UUID" {
		t.Errorf("uuid pgxv5 = %q, want pgtype.UUID", got)
	}

	got = m.GoType(col("uuid", true), noEnums, Options{})
	if got != "uuid.UUID" {
		t.Errorf("uuid NOT NULL = %q, want uuid.UUID", got)
	}

	got = m.GoType(col("uuid", false), noEnums, Options{})
	if got != "uuid.NullUUID" {
		t.Errorf("uuid NULL = %q, want uuid.NullUUID", got)
	}
}

func TestPostgresNetworkTypes(t *testing.T) {
	m := &postgresMapper{}

	got := m.GoType(col("inet", true), noEnums, Options{SqlPackage: "pgx/v5"})
	if got != "netip.Addr" {
		t.Errorf("inet pgxv5 NOT NULL = %q, want netip.Addr", got)
	}

	got = m.GoType(col("inet", false), noEnums, Options{SqlPackage: "pgx/v5"})
	if got != "*netip.Addr" {
		t.Errorf("inet pgxv5 NULL = %q, want *netip.Addr", got)
	}

	got = m.GoType(col("macaddr", true), noEnums, Options{SqlPackage: "pgx/v5"})
	if got != "net.HardwareAddr" {
		t.Errorf("macaddr pgxv5 = %q, want net.HardwareAddr", got)
	}
}

func TestPostgresRangeTypes(t *testing.T) {
	m := &postgresMapper{}

	tests := []struct {
		typ  string
		pkg  string
		want string
	}{
		{"daterange", "pgx/v5", "pgtype.Range[pgtype.Date]"},
		{"daterange", "pgx/v4", "pgtype.Daterange"},
		{"int4range", "pgx/v5", "pgtype.Range[pgtype.Int4]"},
		{"int8range", "pgx/v5", "pgtype.Range[pgtype.Int8]"},
		{"tsrange", "pgx/v5", "pgtype.Range[pgtype.Timestamp]"},
		{"tstzrange", "pgx/v5", "pgtype.Range[pgtype.Timestamptz]"},
		{"numrange", "pgx/v5", "pgtype.Range[pgtype.Numeric]"},
	}

	for _, tt := range tests {
		got := m.GoType(col(tt.typ, true), noEnums, Options{SqlPackage: tt.pkg})
		if got != tt.want {
			t.Errorf("GoType(%q, pkg=%q) = %q, want %q", tt.typ, tt.pkg, got, tt.want)
		}
	}
}

func TestPostgresEnumType(t *testing.T) {
	m := &postgresMapper{}
	enums := []*catalog.Enum{
		{Name: "user_status", Values: []string{"active", "inactive"}},
	}

	got := m.GoType(col("user_status", true), enums, Options{})
	if got != "UserStatus" {
		t.Errorf("enum NOT NULL = %q, want UserStatus", got)
	}

	got = m.GoType(col("user_status", false), enums, Options{})
	if got != "NullUserStatus" {
		t.Errorf("enum NULL = %q, want NullUserStatus", got)
	}
}

func TestPostgresEnumType_NonDefaultSchema(t *testing.T) {
	m := &postgresMapper{}
	enums := []*catalog.Enum{
		{Name: "order_status", Schema: "shop", Values: []string{"pending", "shipped"}},
	}
	opts := Options{DefaultSchema: "public"}

	// NOT NULL enum in non-default schema → prefixed Go type
	got := m.GoType(col("order_status", true), enums, opts)
	if got != "ShopOrderStatus" {
		t.Errorf("non-default schema enum NOT NULL = %q, want ShopOrderStatus", got)
	}

	// NULL enum in non-default schema → Null-prefixed
	got = m.GoType(col("order_status", false), enums, opts)
	if got != "NullShopOrderStatus" {
		t.Errorf("non-default schema enum NULL = %q, want NullShopOrderStatus", got)
	}
}

func TestPostgresEnumType_DefaultSchemaNotPrefixed(t *testing.T) {
	m := &postgresMapper{}
	enums := []*catalog.Enum{
		{Name: "user_status", Schema: "public", Values: []string{"active", "inactive"}},
	}
	opts := Options{DefaultSchema: "public"}

	got := m.GoType(col("user_status", true), enums, opts)
	if got != "UserStatus" {
		t.Errorf("default schema enum = %q, want UserStatus (no prefix)", got)
	}
}

func TestPostgresEnumType_SchemaQualifiedColumnType(t *testing.T) {
	m := &postgresMapper{}
	enums := []*catalog.Enum{
		{Name: "order_status", Schema: "shop", Values: []string{"pending", "shipped"}},
	}
	opts := Options{DefaultSchema: "public"}

	// Column type is schema-qualified (e.g. "shop.order_status")
	got := m.GoType(col("shop.order_status", true), enums, opts)
	if got != "ShopOrderStatus" {
		t.Errorf("schema-qualified column type = %q, want ShopOrderStatus", got)
	}
}

func TestPostgresNullablePointers(t *testing.T) {
	m := &postgresMapper{}
	opts := Options{SqlPackage: "pgx/v5", EmitPointersForNull: true}

	tests := []struct {
		typ  string
		want string
	}{
		{"integer", "*int32"},
		{"bigint", "*int64"},
		{"boolean", "*bool"},
		{"text", "*string"},
		{"float8", "*float64"},
		// Note: pgxv5 driver returns pgtype.* types for timestamp/uuid even with EmitPointersForNull
		{"timestamp", "pgtype.Timestamp"},
		{"uuid", "pgtype.UUID"},
	}

	for _, tt := range tests {
		got := m.GoType(col(tt.typ, false), noEnums, opts)
		if got != tt.want {
			t.Errorf("GoType(%q, NULL, pointers) = %q, want %q", tt.typ, got, tt.want)
		}
	}
}

func TestPostgresByteaType(t *testing.T) {
	m := &postgresMapper{}
	got := m.GoType(col("bytea", true), noEnums, Options{})
	if got != "[]byte" {
		t.Errorf("bytea = %q, want []byte", got)
	}
	got = m.GoType(col("bytea", false), noEnums, Options{})
	if got != "[]byte" {
		t.Errorf("bytea nullable = %q, want []byte", got)
	}
}

func TestPostgresIntervalType(t *testing.T) {
	m := &postgresMapper{}

	got := m.GoType(col("interval", true), noEnums, Options{SqlPackage: "pgx/v5"})
	if got != "pgtype.Interval" {
		t.Errorf("interval pgxv5 = %q, want pgtype.Interval", got)
	}

	got = m.GoType(col("interval", true), noEnums, Options{})
	if got != "int64" {
		t.Errorf("interval default = %q, want int64", got)
	}
}

func TestPostgresHstoreType(t *testing.T) {
	m := &postgresMapper{}

	got := m.GoType(col("hstore", true), noEnums, Options{SqlPackage: "pgx/v5"})
	if got != "pgtype.Hstore" {
		t.Errorf("hstore pgxv5 = %q, want pgtype.Hstore", got)
	}
}

func TestPostgresUnknownType(t *testing.T) {
	m := &postgresMapper{}
	got := m.GoType(col("completely_unknown_type", true), noEnums, Options{})
	if got != "interface{}" {
		t.Errorf("unknown type = %q, want interface{}", got)
	}
}

func TestPostgresVoidAndAny(t *testing.T) {
	m := &postgresMapper{}

	got := m.GoType(col("void", true), noEnums, Options{})
	if got != "interface{}" {
		t.Errorf("void = %q, want interface{}", got)
	}

	got = m.GoType(col("any", true), noEnums, Options{})
	if got != "interface{}" {
		t.Errorf("any = %q, want interface{}", got)
	}
}
