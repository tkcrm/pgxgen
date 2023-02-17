package schema

import (
	"github.com/tkcrm/pgxgen/pkg/sqlc"
	"github.com/tkcrm/pgxgen/pkg/sqlc/cmd"
)

type ISchema interface {
	GetSchema(outputDir string) (cmd.GetCatalogResultItem, error)
}

type schema struct {
	// Key of map is abs schema path (migrations)
	catalogs map[string]cmd.GetCatalogResultItem
}

func New() ISchema {
	return &schema{
		catalogs: make(map[string]cmd.GetCatalogResultItem),
	}
}

func (s *schema) GetSchema(schemaDir string) (cmd.GetCatalogResultItem, error) {
	if item, ok := s.catalogs[schemaDir]; ok {
		return item, nil
	}

	res, err := sqlc.GetCatalogBySchemaDir(schemaDir)
	if err != nil {
		return cmd.GetCatalogResultItem{}, err
	}

	s.catalogs[schemaDir] = res

	return res, nil
}
