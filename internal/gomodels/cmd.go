package gomodels

import (
	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/pkg/logger"
	"github.com/urfave/cli/v2"
)

func CmdFunc(c *cli.Context, l logger.Logger, cfg config.Config) error {
	cfg.CheckErrors(l)
	return New(l, cfg).Generate(c.Context, c.Args().Slice())
}
