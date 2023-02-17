package sqlc

import (
	"os"

	"github.com/tkcrm/pgxgen/pkg/sqlc/cmd"
)

func Run(args []string) int {
	return cmd.Do(args, os.Stdin, os.Stdout, os.Stderr)
}

func GetCatalogs() (cmd.GetCatalogResult, error) {
	return cmd.GetCatalogs()
}

func GetCatalogByOutputDir(outputDir string) (cmd.GetCatalogResultItem, error) {
	return cmd.GetCatalogByOutputDir(outputDir)
}

func GetCatalogBySchemaDir(outputDir string) (cmd.GetCatalogResultItem, error) {
	return cmd.GetCatalogBySchemaDir(outputDir)
}
