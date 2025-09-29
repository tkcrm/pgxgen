package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/crud"
	"github.com/tkcrm/pgxgen/internal/gomodels"
	"github.com/tkcrm/pgxgen/internal/keystone"
	"github.com/tkcrm/pgxgen/internal/sqlc"
	"github.com/tkcrm/pgxgen/internal/typescript"
	"github.com/tkcrm/pgxgen/internal/ver"
	"github.com/tkcrm/pgxgen/pkg/logger"
	"github.com/urfave/cli/v2"
)

var (
	appName = "pgxgen"
	version = "v0.3.12"
)

func getBuildVersion() string {
	return fmt.Sprintf(
		"\nrelease: %s\ngo version: %s",
		version,
		runtime.Version(),
	)
}

func loadConfig(c *cli.Context) (config.Config, error) {
	cf := config.Flags{
		PgxgenConfigFilePath: c.String("pgxgen-config"),
		SqlcConfigFilePath:   c.String("sqlc-config"),
	}

	cfg, err := config.LoadConfig(cf, version)
	if err != nil {
		return cfg, fmt.Errorf("load config error: %w", err)
	}

	return cfg, nil
}

func main() {
	logger := logger.New()

	app := &cli.App{
		Name:    appName,
		Version: getBuildVersion(),
		Usage:   "Generate GO models, DB CRUD, Mobx Keystone models and typescript code based on DDL",
		Suggest: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "pgxgen-config",
				Usage: "Absolute or relative path to pgxgen.yaml file",
				Value: "pgxgen.yaml",
			},
			&cli.StringFlag{
				Name:  "sqlc-config",
				Usage: "Absolute or relative path to sqlc.yaml file",
				Value: "sqlc.yaml",
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "crud",
				Usage: "Generate crud sql's",
				Action: func(c *cli.Context) error {
					cfg, err := loadConfig(c)
					if err != nil {
						return err
					}
					return crud.CmdFunc(c, logger, cfg)
				},
			},
			{
				Name:  "gomodels",
				Usage: "Generate golang models based on existed structs",
				Action: func(c *cli.Context) error {
					cfg, err := loadConfig(c)
					if err != nil {
						return err
					}
					return gomodels.CmdFunc(c, logger, cfg)
				},
			},
			{
				Name:  "keystone",
				Usage: "Generate mobx keystone models",
				Action: func(c *cli.Context) error {
					cfg, err := loadConfig(c)
					if err != nil {
						return err
					}
					return keystone.CmdFunc(c, logger, cfg)
				},
			},
			{
				Name:  "ts",
				Usage: "Generate types for typescript, based on go structs",
				Action: func(c *cli.Context) error {
					cfg, err := loadConfig(c)
					if err != nil {
						return err
					}
					return typescript.CmdFunc(c, logger, cfg)
				},
			},
			{
				Name:  "sqlc",
				Usage: "Generate sqlc code",
				Action: func(c *cli.Context) error {
					cfg, err := loadConfig(c)
					if err != nil {
						return err
					}
					return sqlc.CmdFunc(c, logger, cfg)
				},
			},
			{
				Name:  "update",
				Usage: "Update pgxgen to the latest version",
				Action: func(c *cli.Context) error {
					cfg, err := loadConfig(c)
					if err != nil {
						return err
					}
					return ver.CmdFunc(c, logger, cfg)
				},
			},
			{
				Name:  "version",
				Usage: "Print the version",
				Action: func(c *cli.Context) error {
					fmt.Printf("%s version%s\n", appName, c.App.Version)
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		logger.Fatalf("error: %s", err)
	}
}
