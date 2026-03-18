package sqlc

import (
	"fmt"
	"os"
	"path/filepath"

	sqlcpkg "github.com/sqlc-dev/sqlc/pkg/cli"
	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/pkg/logger"
	"gopkg.in/yaml.v3"
)

// Generator handles sqlc config generation and invocation.
type Generator struct {
	logger    logger.Logger
	configDir string
}

// NewGenerator creates a new sqlc generator.
func NewGenerator(l logger.Logger, configDir string) *Generator {
	return &Generator{
		logger:    l,
		configDir: configDir,
	}
}

// Run generates sqlc.yaml and invokes sqlc for the given schema.
func (g *Generator) Run(schema *config.SchemaConfig) error {
	cfg := BuildSqlcConfig(schema)

	if len(cfg.SQL) == 0 {
		g.logger.Info("no sqlc entries to generate, skipping sqlc")
		return nil
	}

	// Ensure .pgxgen dir exists
	pgxgenDir := filepath.Join(g.configDir, ".pgxgen")
	if err := os.MkdirAll(pgxgenDir, 0o755); err != nil {
		return fmt.Errorf("create .pgxgen dir: %w", err)
	}

	// Write sqlc.yaml
	sqlcPath := filepath.Join(pgxgenDir, "sqlc.yaml")
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal sqlc config: %w", err)
	}

	if err := os.WriteFile(sqlcPath, data, 0o644); err != nil {
		return fmt.Errorf("write sqlc config: %w", err)
	}

	g.logger.Infof("sqlc config written to %s", sqlcPath)

	// Invoke sqlc
	args := []string{"generate", "-f", sqlcPath}
	result := sqlcpkg.Run(args)
	if result != 0 {
		return fmt.Errorf("sqlc generate failed with exit code %d", result)
	}

	g.logger.Info("sqlc generate completed successfully")

	// Post-process: remove models.go, replace imports, replace nullable types
	if schema.Models != nil {
		if err := PostProcess(g.logger, g.configDir, schema, cfg); err != nil {
			return fmt.Errorf("sqlc post-process: %w", err)
		}
	}

	return nil
}
