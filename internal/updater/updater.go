package updater

import (
	"context"
	"fmt"

	"github.com/tkcrm/pgxgen/internal/ver"
	"github.com/tkcrm/pgxgen/pkg/logger"
)

// CheckAndUpdate checks for a new version and updates if available.
func CheckAndUpdate(ctx context.Context, l logger.Logger, currentVersion string) error {
	resp, err := ver.CheckAndUpdateVersion(ctx, currentVersion)
	if err != nil {
		return fmt.Errorf("check latest release version error: %w", err)
	}

	if resp != nil && !resp.IsLatest {
		l.Info(resp.Message)
	} else {
		l.Info("Congratulations! You are using the latest version")
	}

	return nil
}
