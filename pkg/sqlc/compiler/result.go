package compiler

import (
	"github.com/tkcrm/pgxgen/pkg/sqlc/sql/catalog"
)

type Result struct {
	Catalog *catalog.Catalog
	Queries []*Query
}
