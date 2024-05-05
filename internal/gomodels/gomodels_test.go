package gomodels_test

import (
	"context"
	"os"
	"testing"

	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/generator"
	"github.com/tkcrm/pgxgen/internal/gomodels"
	"github.com/tkcrm/pgxgen/pkg/logger"
)

func initGoModels(t *testing.T) generator.IGenerator {
	logger := logger.New()

	cfg, err := config.LoadTestConfig("../../testdata/gomodels/")
	if err != nil {
		t.Fatal(err)
	}

	return gomodels.New(logger, cfg)
}

func Test_Gomodels(t *testing.T) {
	gm := initGoModels(t)

	ctx := context.Background()
	if err := gm.Generate(ctx, os.Args[1:]); err != nil {
		t.Fatal(err)
	}
}
