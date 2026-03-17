package genmodels

import (
	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/pkg/logger"
	"github.com/urfave/cli/v2"
)

func CmdFunc(_ *cli.Context, l logger.Logger, cfg config.Config) error {
	g := New(l, cfg)
	return g.Generate()
}
