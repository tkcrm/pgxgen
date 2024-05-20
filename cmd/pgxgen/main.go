package main

import (
	"os"

	"github.com/cristalhq/acmd"
	"github.com/cristalhq/flagx"
	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/crud"
	"github.com/tkcrm/pgxgen/internal/gomodels"
	"github.com/tkcrm/pgxgen/internal/keystone"
	"github.com/tkcrm/pgxgen/internal/sqlc"
	"github.com/tkcrm/pgxgen/internal/typescript"
	"github.com/tkcrm/pgxgen/internal/ver"
	"github.com/tkcrm/pgxgen/pkg/logger"
)

var (
	appName = "pgxgen"
	version = "v0.3.4"
)

func main() {
	logger := logger.New()

	// custom config file path
	var cf config.Flags
	fset := flagx.NewFlagSet(appName, os.Stdout)
	fset.String(&cf.PgxgenConfigFilePath, "pgxgen-config", "", "pgxgen.yaml", "absolute or relative path to sqlc.yaml file")
	fset.String(&cf.SqlcConfigFilePath, "sqlc-config", "", "sqlc.yaml", "absolute or relative path to pgxgen.yaml file")

	// parse flags
	if err := fset.Parse(os.Args[1:]); err != nil {
		logger.Fatalf("failed to parse flags: %s", err)
	}

	cfg, err := config.LoadConfig(cf, version)
	if err != nil {
		logger.Fatalf("load config error: %s", err)
	}

	cmds := []acmd.Command{
		{
			Name:        "crud",
			Description: "generate crud sql's",
			ExecFunc:    crud.CmdFunc(logger, cfg),
		},
		{
			Name:        "gomodels",
			Description: "generate golang models based on existed structs",
			ExecFunc:    gomodels.CmdFunc(logger, cfg),
		},
		{
			Name:        "keystone",
			Description: "generate mobx keystone models",
			ExecFunc:    keystone.CmdFunc(logger, cfg),
		},
		{
			Name:        "ts",
			Description: "generate types for typescript, based on go structs",
			ExecFunc:    typescript.CmdFunc(logger, cfg),
		},
		{
			Name:        "sqlc",
			Description: "generate sqlc code",
			ExecFunc:    sqlc.CmdFunc(logger, cfg),
		},
		{
			Name:        "check-version",
			Description: "check for new version",
			ExecFunc:    ver.CmdFunc(logger, cfg),
		},
	}

	r := acmd.RunnerOf(cmds, acmd.Config{
		AppName:         "pgxgen",
		AppDescription:  "Generate GO models, DB CRUD, Mobx Keystone models and typescript code based on DDL",
		PostDescription: "pgxgen crud",
		Version:         version,
		Args:            append([]string{os.Args[0]}, fset.Args()...),
	})

	if err := r.Run(); err != nil {
		logger.Fatalf("error: %s", err)
	}
}
