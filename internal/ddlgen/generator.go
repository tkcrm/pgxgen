package ddlgen

import (
	"fmt"

	"github.com/tkcrm/pgxgen/internal/sqlparser/catalog"
)

// Generator converts a Catalog to consolidated DDL SQL text.
type Generator interface {
	Generate(cat *catalog.Catalog) (string, error)
}

// New creates a DDL generator for the given database engine.
func New(engine string) (Generator, error) {
	switch engine {
	case "postgresql":
		return &postgresGenerator{}, nil
	case "sqlite":
		return &sqliteGenerator{}, nil
	default:
		return nil, fmt.Errorf("ddlgen: unsupported engine %q", engine)
	}
}
