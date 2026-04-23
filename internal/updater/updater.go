package updater

import (
	"context"
	"fmt"

	"github.com/tkcrm/pgxgen/internal/ver"
	"github.com/tkcrm/pgxgen/pkg/logger"
)

// CheckAndUpdate checks for a new version and updates if available.
func CheckAndUpdate(ctx context.Context, l logger.Logger, currentVersion string) error {
	if err := ver.CheckAndUpdateVersion(ctx, l, currentVersion); err != nil {
		return fmt.Errorf("check latest release version error: %w", err)
	}

	return nil
}
