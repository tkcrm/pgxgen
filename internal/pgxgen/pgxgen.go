package pgxgen

import (
	"fmt"
	"os"

	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/crud"
	"github.com/tkcrm/pgxgen/internal/generator"
	"github.com/tkcrm/pgxgen/internal/gomodels"
	"github.com/tkcrm/pgxgen/internal/keystone"
	"github.com/tkcrm/pgxgen/internal/sqlc"
	"github.com/tkcrm/pgxgen/internal/typescript"
	"gopkg.in/yaml.v3"
)

var version = "v0.0.25"

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

	c.Pgxgen.Version = version

	var generator generator.IGenerator
	switch args[0] {
	case "version":
		fmt.Printf("%s\n", version)
	case "crud":
		generator = crud.New(c)
	case "gomodels":
		generator = gomodels.New(c)
	case "keystone":
		generator = keystone.New(c)
	case "ts":
		generator = typescript.New(c)
	case "sqlc":
		generator = sqlc.New(c)
	default:
		return fmt.Errorf("undefined command %s", args[0])
	}

	if generator != nil {
		if err := generator.Generate(args); err != nil {
			return err
		}
	}

	// Check latest version
	// versionResponse, err := ver.CheckLastestReleaseVersion(version)
	// if err != nil {
	// 	fmt.Printf("check latest release version error: %v\n", err)
	// }

	// if versionResponse != nil && !versionResponse.IsLatest {
	// 	fmt.Println(versionResponse.Message)
	// }

	return nil
}
