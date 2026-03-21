package catalog

// Catalog represents the parsed database schema.
type Catalog struct {
	DefaultSchema string
	Schemas       []*Schema
}

// Schema represents a database schema containing tables, views, and types.
type Schema struct {
	Name       string
	Tables     []*Table
	Views      []*View
	Enums      []*Enum
	Extensions []string
}

// Table represents a database table.
type Table struct {
	Name        string
	Schema      string
	Columns     []*Column
	Comment     string
	PrimaryKey  *PrimaryKey
	Indexes     []*Index
	ForeignKeys []*ForeignKey
	Uniques     []*UniqueConstraint
	Checks      []*CheckConstraint
}

// View represents a database view.
type View struct {
	Name    string
	Schema  string
	Columns []*Column
	Comment string
	Query   string // the SELECT statement defining the view
}

// Column represents a column in a database table.
type Column struct {
	Name       string
	Type       string // raw SQL type name, e.g. "UUID", "VARCHAR(255)", "TIMESTAMP"
	FullType   string // full SQL type with modifiers, e.g. "character varying(255)", "uuid"
	NotNull    bool
	IsArray    bool
	ArrayDims  int
	Comment    string
	Length     *int
	IsUnsigned bool
	Default    string // raw SQL expression, e.g. "uuid_generate_v4()", "CURRENT_TIMESTAMP"
	IsPrimary  bool
}

// PrimaryKey represents a PRIMARY KEY constraint.
type PrimaryKey struct {
	Name    string
	Columns []string
}

// Index represents a database index.
type Index struct {
	Name        string
	Columns     []string
	IsUnique    bool
	Where       string // partial index predicate, raw SQL
	IfNotExists bool
}

// ForeignKey represents a FOREIGN KEY constraint.
type ForeignKey struct {
	Name       string
	Columns    []string
	RefTable   string
	RefSchema  string
	RefColumns []string
	OnDelete   string
	OnUpdate   string
}

// UniqueConstraint represents a UNIQUE constraint.
type UniqueConstraint struct {
	Name    string
	Columns []string
}

// CheckConstraint represents a CHECK constraint.
type CheckConstraint struct {
	Name       string
	Expression string
}

// Enum represents a database enum type.
type Enum struct {
	Name   string
	Values []string
}
