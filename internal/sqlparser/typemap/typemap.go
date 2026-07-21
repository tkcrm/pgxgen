package typemap

import (
	"fmt"
	"strings"

	"github.com/tkcrm/pgxgen/internal/sqlparser/catalog"
)

// Options configures Go type generation.
type Options struct {
	SqlPackage          string // "pgx/v5", "pgx/v4", "database/sql"
	EmitPointersForNull bool
	DefaultSchema       string // default database schema (e.g. "public" for PostgreSQL)
}

// TypeMapper maps SQL column types to Go types.
type TypeMapper interface {
	GoType(col *catalog.Column, enums []*catalog.Enum, opts Options) string
}

// NewTypeMapper creates a type mapper for the given database engine.
func NewTypeMapper(engine string) (TypeMapper, error) {
	switch engine {
	case "postgresql", "":
		return &postgresMapper{}, nil
	case "mysql":
		return &mysqlMapper{}, nil
	case "sqlite":
		return &sqliteMapper{}, nil
	default:
		return nil, fmt.Errorf("unsupported engine: %s", engine)
	}
}

type sqlDriver int

const (
	driverDatabaseSQL sqlDriver = iota
	driverPGXV4
	driverPGXV5
	driverLibPQ
)

func parseDriver(pkg string) sqlDriver {
	switch pkg {
	case "pgx/v4":
		return driverPGXV4
	case "pgx/v5":
		return driverPGXV5
	default:
		return driverDatabaseSQL
	}
}

// structName converts a SQL name to a Go exported name (CamelCase).
func structName(name string) string {
	var out strings.Builder
	for p := range strings.SplitSeq(name, "_") {
		if p == "" {
			continue
		}
		out.WriteString(strings.ToUpper(p[:1]))
		out.WriteString(p[1:])
	}
	return out.String()
}
