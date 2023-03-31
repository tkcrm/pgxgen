package typescript

import (
	"context"

	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/pkg/logger"
)

func CmdFunc(l logger.Logger, cfg config.Config) func(ctx context.Context, args []string) error {
	return func(ctx context.Context, args []string) error {
		return New(l, cfg).Generate(args)
	}
}
