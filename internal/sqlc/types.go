package sqlc

import "github.com/tkcrm/pgxgen/internal/config"

type replaceFunc func(c config.Config, str string) (string, error)
