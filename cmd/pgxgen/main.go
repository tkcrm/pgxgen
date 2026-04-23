package main

import (
	"context"
	"os"

	pgxcli "github.com/tkcrm/pgxgen/internal/cli"
	"github.com/tkcrm/pgxgen/pkg/logger"
)

var version = "v0.5.6"

func main() {
	l := logger.New()

	app := pgxcli.NewApp(version)
	if err := app.Run(context.Background(), os.Args); err != nil {
		l.Fatalf("error: %s", err)
	}
}
