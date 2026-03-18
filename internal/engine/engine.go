package engine

import "fmt"

// Engine abstracts database-specific SQL dialect differences.
type Engine interface {
	// Name returns the engine name (postgresql, mysql, sqlite).
	Name() string

	// ParamPlaceholder returns the parameter placeholder for the given index (1-based).
	// PostgreSQL: $1, $2, $3
	// MySQL/SQLite: ?, ?, ?
	ParamPlaceholder(index int) string

	// BoolCast wraps an expression with a boolean cast if needed.
	// PostgreSQL: expr::boolean
	// MySQL/SQLite: expr (no cast)
	BoolCast(expr string) string

	// TimestampFunc returns the current timestamp function.
	// PostgreSQL: now()
	// MySQL: NOW()
	// SQLite: datetime('now')
	TimestampFunc() string

	// SupportsReturning returns true if the engine supports RETURNING clause.
	SupportsReturning() bool

	// BatchInsertStrategy returns the batch insert strategy.
	// PostgreSQL: copyfrom (uses pgx CopyFrom)
	// MySQL/SQLite: multi-values
	BatchInsertStrategy() string
}

// New creates an Engine for the given engine name.
func New(name string) (Engine, error) {
	switch name {
	case "postgresql":
		return &postgresql{}, nil
	case "mysql":
		return &mysql{}, nil
	case "sqlite":
		return &sqlite{}, nil
	default:
		return nil, fmt.Errorf("unsupported engine: %s", name)
	}
}
