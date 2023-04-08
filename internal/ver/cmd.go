package ver

import (
	"context"
	"fmt"

	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/pkg/logger"
)

func CmdFunc(l logger.Logger, cfg config.Config) func(ctx context.Context, args []string) error {
	return func(ctx context.Context, args []string) error {
		resp, err := CheckLastestReleaseVersion(ctx, cfg.Pgxgen.Version)
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
}
