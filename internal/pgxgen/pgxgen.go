package pgxgen

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	sqlc "github.com/kyleconroy/sqlc/pkg/cli"
	"github.com/tkcrm/pgxgen/internal/config"
	"golang.org/x/tools/imports"
	"gopkg.in/yaml.v3"
)

var version = "0.0.4"

func Start(args []string) error {

	if len(args) == 0 {
		fmt.Println(helpMessage)
		return nil
	}

	var sqlcConfig config.SqlcConfig
	sqlcConfigFile, err := os.ReadFile("sqlc.yaml")
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(sqlcConfigFile, &sqlcConfig); err != nil {
		return err
	}

	for _, p := range sqlcConfig.Packages {
		if p.Path == "" {
			return fmt.Errorf("undefined path in sqlc.yaml")
		}
		if p.Queries == "" {
			return fmt.Errorf("undefined queries in sqlc.yaml")
		}
	}

	var pgxgenConfig config.PgxgenConfig
	pgxgenConfigFile, err := os.ReadFile("pgxgen.yaml")
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(pgxgenConfigFile, &pgxgenConfig); err != nil {
		return err
	}

	c := config.Config{
		Sqlc:   sqlcConfig,
		Pgxgen: pgxgenConfig,
	}

	switch args[0] {
	case "version":
		fmt.Printf("Tool version: %s;\nGo version: %s\n", version, runtime.Version())
	case "gencrud":
		if err := genCRUD(args, c); err != nil {
			return err
		}
		fmt.Println("crud successfully generated")
	case "sqlc":
		if err := processSqlc(args[1:], c); err != nil {
			return err
		}
		fmt.Println("sqlc successfully generated")
	default:
		return fmt.Errorf("undefined command %s", args[0])
	}

	return nil
}

func processSqlc(args []string, c config.Config) error {

	genResult := sqlc.Run(args)
	if genResult != 0 {
		return nil
	}

	for _, p := range c.Sqlc.Packages {

		files, err := os.ReadDir(p.Path)
		if err != nil {
			return err
		}

		for _, file := range files {
			r := regexp.MustCompile(`(\.go)`)
			if r.MatchString(file.Name()) {
				if err := replaceStructTypes(filepath.Join(p.Path, file.Name())); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func replaceStructTypes(path string) error {

	models, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	result := string(models)
	for old, new := range types {
		result = strings.ReplaceAll(result, old, new)
	}

	formated, err := imports.Process(path, []byte(result), nil)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(path), formated, 0644); err != nil {
		return err
	}

	return nil
}
