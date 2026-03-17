package sqlparser

import (
	"fmt"

	"github.com/tkcrm/pgxgen/internal/sqlparser/catalog"
)

// Parser parses SQL schema files and returns a catalog.
type Parser interface {
	ParseSchema(files []string) (*catalog.Catalog, error)
}

// NewParser creates a parser for the given database engine.
func NewParser(engine string) (Parser, error) {
	switch engine {
	case "postgresql", "":
		return newPostgresParser(), nil
	case "mysql":
		return newMysqlParser(), nil
	case "sqlite":
		return newSqliteParser(), nil
	default:
		return nil, fmt.Errorf("unsupported engine: %s", engine)
	}
}
