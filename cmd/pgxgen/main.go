package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

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
	"gopkg.in/yaml.v2"
)

var version = "v0.0.27"

func main() {
	logger := logger.New()

	cfg, err := loadConfig()
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

type configFlags struct {
	sqlcConfigFilePath   string
	pgxgenConfigFilePath string
}

func (c *configFlags) Flags() *flag.FlagSet {
	fset := flagx.NewFlagSet("pgxgen config", os.Stderr)
	fset.String(&c.sqlcConfigFilePath, "sqlc-config", "", "sqlc.yaml", "absolute or relative path to sqlc.yaml file")
	fset.String(&c.pgxgenConfigFilePath, "pgxgen-config", "", "pgxgen.yaml", "absolute or relative path to pgxgen.yaml file")
	return fset.AsStdlib()
}

// loadConfig return common config with sqlc and pgxgen data
func loadConfig() (config.Config, error) {
	var cfg config.Config

	// parse config flags
	var cf configFlags
	if err := cf.Flags().Parse(os.Args[1:]); err != nil {
		return cfg, fmt.Errorf("failed to parse flags: %w", err)
	}

	// load sqlc config
	var sqlcConfig config.Sqlc
	sqlcConfigFile, err := os.ReadFile(cf.sqlcConfigFilePath)
	if err != nil {
		return cfg, fmt.Errorf("failed to read sqlc config file: %w", err)
	}

	if err := yaml.Unmarshal(sqlcConfigFile, &sqlcConfig); err != nil {
		return cfg, err
	}

	// validate sqlc config
	for _, p := range sqlcConfig.Packages {
		if p.Path == "" {
			return cfg, fmt.Errorf("undefined path in sqlc.yaml")
		}
		if p.Queries == "" {
			return cfg, fmt.Errorf("undefined queries in sqlc.yaml")
		}
	}

	// load pgxgen config
	var pgxgenConfig config.Pgxgen
	pgxgenConfigFile, err := os.ReadFile(cf.pgxgenConfigFilePath)
	if err != nil {
		return cfg, err
	}

	if err := yaml.Unmarshal(pgxgenConfigFile, &pgxgenConfig); err != nil {
		return cfg, err
	}

	cfg.Sqlc = sqlcConfig
	cfg.Pgxgen = pgxgenConfig
	cfg.Pgxgen.Version = version

	return cfg, nil
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
