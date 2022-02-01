package pgxgen

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	sqlc "github.com/kyleconroy/sqlc/pkg/cli"
	"github.com/tkcrm/pgxgen/internal/config"
	"golang.org/x/tools/imports"
	"gopkg.in/yaml.v3"
)

func Start(args []string) error {

	if len(args) == 0 {
		fmt.Println(helpMessage)
		return nil
	}

	var c config.SqlcConfig

	configFile, err := os.ReadFile("sqlc.yaml")
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(configFile, &c); err != nil {
		return err
	}

	switch args[0] {
	case "gencrud":
		if err := processGenCRUD(args, c); err != nil {
			return err
		}
	default:
		if err := processSqlc(args, c); err != nil {
			return err
		}
	}

	fmt.Println("successfully generated")

	return nil
}

func processSqlc(args []string, c config.SqlcConfig) error {

	genResult := sqlc.Run(args)
	if genResult != 0 {
		os.Exit(genResult)
	}

	for _, p := range c.Packages {
		if err := processModels(fmt.Sprintf("%s/models.go", p.Path)); err != nil {
			log.Fatal(err)
		}
	}

	return nil
}

func processModels(path string) error {

	models, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	result := string(models)
	for old, new := range replaceTypes {
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
