package codegen

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/pkg/logger"
)

const testSchemaSQL = `CREATE TABLE users (
  id TEXT PRIMARY KEY,
  email TEXT NOT NULL,
  name TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

// setupTestProject writes a minimal SQL schema to a temp dir and returns
// (configDir, configPath). The configPath file itself is not written —
// tests construct V2Config in code and pass configPath only so the
// orchestrator can derive configDir from it.
func setupTestProject(t *testing.T) (string, string) {
	t.Helper()
	tmp := t.TempDir()
	migrationsDir := filepath.Join(tmp, "sql", "migrations")
	require.NoError(t, os.MkdirAll(migrationsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(migrationsDir, "001_init.sql"), []byte(testSchemaSQL), 0o644))
	return tmp, filepath.Join(tmp, "pgxgen.yaml")
}

func newTestConfig(withSqlc bool) *config.V2Config {
	cfg := &config.V2Config{
		Version: "2",
		Schemas: []config.SchemaConfig{
			{
				Name:      "main",
				Engine:    "sqlite",
				SchemaDir: "sql/migrations",
				Defaults: &config.DefaultsConfig{
					QueriesDirPrefix: "sql/queries",
					OutputDirPrefix:  "internal/store/repos",
					Crud: &config.CrudDefaultsConfig{
						Methods: map[string]*config.MethodConfig{
							"get":    {},
							"delete": {},
						},
					},
				},
				Tables: map[string]config.TableConfig{
					"users": {
						PrimaryColumn: "id",
						Crud: &config.TableCrudConfig{
							Methods: map[string]*config.MethodConfig{
								"get":    {},
								"delete": {},
							},
						},
					},
				},
			},
		},
	}
	if withSqlc {
		cfg.Schemas[0].Sqlc = &config.SqlcConfig{
			Defaults: &config.SqlcDefaultsConfig{
				SqlPackage:    "database/sql",
				EmitInterface: true,
			},
		}
	}
	return cfg
}

// TestGenerate_CrudWrittenBeforeSqlc is the regression test for the bug
// where sqlc was invoked before CRUD SQL files reached disk. With the
// fix, Generate writes CRUD results before running sqlc, so sqlc finds
// its inputs on a fresh project (no pre-existing query directories).
func TestGenerate_CrudWrittenBeforeSqlc(t *testing.T) {
	tmp, configPath := setupTestProject(t)
	cfg := newTestConfig(true)

	queriesPath := filepath.Join(tmp, "sql", "queries", "users")
	require.NoDirExists(t, queriesPath, "queries dir must not pre-exist — this is the fresh-project scenario")

	orch := NewOrchestrator(logger.New(), cfg, configPath)
	_, err := orch.Generate(context.Background(), GenerateOpts{})
	require.NoError(t, err, "Generate must succeed on a fresh project")

	assert.FileExists(t, filepath.Join(queriesPath, "users_gen.sql"), "CRUD SQL must be written")

	sqlcOutDir := filepath.Join(tmp, "internal", "store", "repos", "users")
	assert.DirExists(t, sqlcOutDir, "sqlc must have produced its output directory")

	entries, err := os.ReadDir(sqlcOutDir)
	require.NoError(t, err)
	var goFiles []string
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".go" {
			goFiles = append(goFiles, e.Name())
		}
	}
	assert.NotEmpty(t, goFiles, "sqlc must have generated at least one .go file in %s", sqlcOutDir)
}

// TestGenerate_DryRunSkipsWritesAndSqlc verifies that dry-run touches
// nothing on disk and does not invoke sqlc (which has no preview mode).
func TestGenerate_DryRunSkipsWritesAndSqlc(t *testing.T) {
	tmp, configPath := setupTestProject(t)
	cfg := newTestConfig(true)

	orch := NewOrchestrator(logger.New(), cfg, configPath)
	_, err := orch.Generate(context.Background(), GenerateOpts{DryRun: true})
	require.NoError(t, err)

	assert.NoDirExists(t, filepath.Join(tmp, "sql", "queries"), "dry-run must not write CRUD files")
	assert.NoDirExists(t, filepath.Join(tmp, "internal"), "dry-run must not invoke sqlc")
	assert.NoDirExists(t, filepath.Join(tmp, ".pgxgen"), "dry-run must not write sqlc.yaml")
}

// TestGenerate_CrudOnlyTargetSkipsSqlc verifies the crud-only target
// writes CRUD files and does not invoke sqlc.
func TestGenerate_CrudOnlyTargetSkipsSqlc(t *testing.T) {
	tmp, configPath := setupTestProject(t)
	cfg := newTestConfig(true)

	orch := NewOrchestrator(logger.New(), cfg, configPath)
	_, err := orch.Generate(context.Background(), GenerateOpts{Targets: []string{"crud"}})
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(tmp, "sql", "queries", "users", "users_gen.sql"))
	assert.NoDirExists(t, filepath.Join(tmp, "internal"), "sqlc must not run when target is crud only")
	assert.NoFileExists(t, filepath.Join(tmp, ".pgxgen", "sqlc.yaml"))
}

// TestGenerate_SqlcConfigRemovedByDefault verifies that after a
// successful run the generated .pgxgen/sqlc.yaml and its directory are
// cleaned up (keep_generated_config defaults to false).
func TestGenerate_SqlcConfigRemovedByDefault(t *testing.T) {
	tmp, configPath := setupTestProject(t)
	cfg := newTestConfig(true)

	orch := NewOrchestrator(logger.New(), cfg, configPath)
	_, err := orch.Generate(context.Background(), GenerateOpts{})
	require.NoError(t, err)

	assert.NoFileExists(t, filepath.Join(tmp, ".pgxgen", "sqlc.yaml"), "sqlc.yaml must be removed after run")
	assert.NoDirExists(t, filepath.Join(tmp, ".pgxgen"), "empty .pgxgen dir must be removed after run")
}

// TestGenerate_SqlcConfigKeptWhenFlagSet verifies that setting
// keep_generated_config: true preserves .pgxgen/sqlc.yaml after run.
func TestGenerate_SqlcConfigKeptWhenFlagSet(t *testing.T) {
	tmp, configPath := setupTestProject(t)
	cfg := newTestConfig(true)
	cfg.Schemas[0].Sqlc.KeepGeneratedConfig = true

	orch := NewOrchestrator(logger.New(), cfg, configPath)
	_, err := orch.Generate(context.Background(), GenerateOpts{})
	require.NoError(t, err)

	sqlcPath := filepath.Join(tmp, ".pgxgen", "sqlc.yaml")
	assert.FileExists(t, sqlcPath, "sqlc.yaml must be kept when keep_generated_config=true")
	data, err := os.ReadFile(sqlcPath)
	require.NoError(t, err)
	assert.NotEmpty(t, data, "kept sqlc.yaml must have content")
}
