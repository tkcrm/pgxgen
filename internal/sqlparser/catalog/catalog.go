package catalog

// Catalog represents the parsed database schema.
type Catalog struct {
	DefaultSchema string
	Schemas       []*Schema
}

// Schema represents a database schema containing tables and types.
type Schema struct {
	Name   string
	Tables []*Table
	Enums  []*Enum
}

// Table represents a database table.
type Table struct {
	Name    string
	Schema  string
	Columns []*Column
	Comment string
}

// Column represents a column in a database table.
type Column struct {
	Name       string
	Type       string // raw SQL type name, e.g. "UUID", "VARCHAR(255)", "TIMESTAMP"
	NotNull    bool
	IsArray    bool
	ArrayDims  int
	Comment    string
	Length     *int
	IsUnsigned bool
}

// Enum represents a database enum type.
type Enum struct {
	Name   string
	Values []string
}
