package pgxgen

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	sqlc "github.com/kyleconroy/sqlc/pkg/cli"
	"github.com/tkcrm/pgxgen/internal/config"
	"golang.org/x/tools/imports"
	"gopkg.in/yaml.v3"
)

var version = "0.0.7"

func Start(args []string) error {

	if len(args) == 0 {
		fmt.Println(helpMessage)
		return nil
	}

	var sqlcConfig config.Sqlc
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

	var pgxgenConfig config.Pgxgen
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
		fmt.Printf("%s\n", version)
	case "gencrud":
		if err := generateCRUD(args, c); err != nil {
			return err
		}
		fmt.Println("crud successfully generated")
	case "genmodels":
		if err := generateModels(args, c); err != nil {
			return err
		}
		fmt.Println("models successfully generated")
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

		modelFileName := p.OutputModelsFileName
		if modelFileName == "" {
			modelFileName = "models.go"
		}

		for _, file := range files {
			r := regexp.MustCompile(`(\.go)`)
			if r.MatchString(file.Name()) {
				if err := replace(c, filepath.Join(p.Path, file.Name()), replaceStructTypes); err != nil {
					return err
				}
			}

			if file.Name() == modelFileName {
				if err := replace(c, filepath.Join(p.Path, file.Name()), replaceJsonTags); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func replaceStructTypes(c config.Config, str string) string {
	res := str
	for old, new := range types {
		res = strings.ReplaceAll(res, old, new)
	}
	return res
}

func replaceJsonTags(c config.Config, str string) string {
	res := str
	for _, field := range c.Pgxgen.JsonTags.Omitempty {
		res = strings.ReplaceAll(res, fmt.Sprintf("json:\"%s\"", field), fmt.Sprintf("json:\"%s,omitempty\"", field))
	}
	for _, field := range c.Pgxgen.JsonTags.Hide {
		res = strings.ReplaceAll(res, fmt.Sprintf("json:\"%s\"", field), "json:\"-\"")
	}
	return res
}

func replace(c config.Config, path string, fn func(c config.Config, str string) string) error {

	file, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	result := fn(c, string(file))

	formated, err := imports.Process(path, []byte(result), nil)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(path), formated, os.ModePerm); err != nil {
		return err
	}

	return nil
}
