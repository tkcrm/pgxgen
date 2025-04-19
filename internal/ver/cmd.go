package ver

import (
	"fmt"

	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/pkg/logger"
	"github.com/urfave/cli/v2"
)

func CmdFunc(c *cli.Context, l logger.Logger, cfg config.Config) error {
	resp, err := CheckAndUpdateVersion(c.Context, cfg.Pgxgen.Version)
	if err != nil {
		return fmt.Errorf("check latest release version error: %s", err)
	}

	if resp != nil && !resp.IsLatest {
		l.Info(resp.Message)
	} else {
		l.Info("Congratulations! You are using the latest version")
	}

	return nil
}
