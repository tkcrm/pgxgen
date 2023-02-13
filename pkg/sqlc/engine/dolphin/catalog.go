package dolphin

import (
	"github.com/tkcrm/pgxgen/pkg/sqlc/sql/catalog"
)

func NewCatalog() *catalog.Catalog {
	def := "public" // TODO: What is the default database for MySQL?
	return &catalog.Catalog{
		DefaultSchema: def,
		Schemas: []*catalog.Schema{
			defaultSchema(def),
		},
		Extensions: map[string]struct{}{},
	}
}
