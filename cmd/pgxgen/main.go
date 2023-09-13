package main

import (
	"os"
	"strings"

	"github.com/cristalhq/acmd"
	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/crud"
	"github.com/tkcrm/pgxgen/internal/gomodels"
	"github.com/tkcrm/pgxgen/internal/keystone"
	"github.com/tkcrm/pgxgen/internal/sqlc"
	"github.com/tkcrm/pgxgen/internal/typescript"
	"github.com/tkcrm/pgxgen/internal/ver"
	"github.com/tkcrm/pgxgen/pkg/logger"
)

var version = "v0.2.6"

func main() {
	logger := logger.New()

	cfg, err := config.LoadConfig(version)
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
		Args:            filterFlags(),
	})

	if err := r.Run(); err != nil {
		logger.Fatalf("error: %s", err)
	}
}

// filterFlags filter os.Args from `pgxgen-config` and `sqlc-config` flags
func filterFlags() []string {
	res := []string{}
	args := os.Args
	for i := 0; i < len(args); i++ {
		if strings.Contains(args[i], "pgxgen-config") ||
			strings.Contains(args[i], "sqlc-config") {
			i++
			continue
		}
		res = append(res, args[i])
	}

	return res
}
