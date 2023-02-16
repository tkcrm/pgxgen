package schema

import (
	"github.com/tkcrm/pgxgen/pkg/sqlc"
	"github.com/tkcrm/pgxgen/pkg/sqlc/cmd"
)

func GetTableMetaData(outputDir string) (cmd.GetCatalogResultItem, error) {
	return sqlc.GetCatalogByOutputDir(outputDir)
}
