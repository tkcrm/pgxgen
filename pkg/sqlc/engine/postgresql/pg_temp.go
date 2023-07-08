package postgresql

import (
	"github.com/tkcrm/pgxgen/pkg/sqlc/sql/catalog"
)

func pgTemp() *catalog.Schema {
	return &catalog.Schema{Name: "pg_temp"}
}
