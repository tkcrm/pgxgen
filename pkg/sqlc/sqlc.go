package sqlc

import (
	"os"

	"github.com/tkcrm/pgxgen/pkg/sqlc/cmd"
)

func Run(args []string) int {
	return cmd.Do(args, os.Stdin, os.Stdout, os.Stderr)
}

func GetCatalogs(configFilePath string) (cmd.GetCatalogResult, error) {
	return cmd.GetCatalogs(configFilePath)
}

func GetCatalogByOutputDir(catalogs cmd.GetCatalogResult, outputDir string) (cmd.GetCatalogResultItem, error) {
	return cmd.GetCatalogByOutputDir(catalogs, outputDir)
}

func GetCatalogBySchemaDir(configFilePath, outputDir string) (cmd.GetCatalogResultItem, error) {
	return cmd.GetCatalogBySchemaDir(configFilePath, outputDir)
}
