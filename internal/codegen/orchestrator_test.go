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

// TestGenerate_SchemaQualifiedTableNames verifies that when table keys
// contain a schema prefix (e.g. "myschema.users"), CRUD SQL uses the
// schema-qualified name in SQL statements and schema-prefixed Go name for paths.
func TestGenerate_SchemaQualifiedTableNames(t *testing.T) {
	tmp := t.TempDir()
	migrationsDir := filepath.Join(tmp, "sql", "migrations")
	require.NoError(t, os.MkdirAll(migrationsDir, 0o755))

	schemaSQL := `CREATE SCHEMA myschema;
CREATE TABLE myschema.users (
  id TEXT PRIMARY KEY,
  email TEXT NOT NULL,
  name TEXT NOT NULL
);`
	require.NoError(t, os.WriteFile(filepath.Join(migrationsDir, "001_init.sql"), []byte(schemaSQL), 0o644))

	configPath := filepath.Join(tmp, "pgxgen.yaml")
	cfg := &config.V2Config{
		Version: "2",
		Schemas: []config.SchemaConfig{
			{
				Name:      "main",
				Engine:    "postgresql",
				SchemaDir: "sql/migrations",
				Defaults: &config.DefaultsConfig{
					QueriesDirPrefix: "sql/queries",
					OutputDirPrefix:  "internal/store/repos",
				},
				Tables: map[string]config.TableConfig{
					"myschema.users": {
						PrimaryColumn: "id",
						Crud: &config.TableCrudConfig{
							Methods: map[string]*config.MethodConfig{
								"get":    {},
								"create": {Returning: "*"},
							},
						},
					},
				},
			},
		},
	}

	orch := NewOrchestrator(logger.New(), cfg, configPath)
	_, err := orch.Generate(context.Background(), GenerateOpts{Targets: []string{"crud"}})
	require.NoError(t, err)

	// File should use schema-prefixed Go name for path
	sqlPath := filepath.Join(tmp, "sql", "queries", "myschema_users", "myschema_users_gen.sql")
	assert.FileExists(t, sqlPath, "CRUD SQL file should use schema-prefixed Go name for path")

	// Schema-qualified dot path should NOT exist
	assert.NoFileExists(t, filepath.Join(tmp, "sql", "queries", "myschema.users", "myschema.users_gen.sql"))
	// Bare name path should NOT exist either (schema prefix required)
	assert.NoFileExists(t, filepath.Join(tmp, "sql", "queries", "users", "users_gen.sql"))

	// SQL content should use schema-qualified name
	data, err := os.ReadFile(sqlPath)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "myschema.users", "SQL should contain schema-qualified table name")
	assert.Contains(t, content, "INSERT INTO myschema.users", "INSERT should use schema-qualified name")
	assert.Contains(t, content, "SELECT * FROM myschema.users", "SELECT should use schema-qualified name")

	// Method names should use schema-prefixed Go name
	assert.Contains(t, content, "CreateMyschemaUser", "method name should use schema-prefixed name")
	assert.Contains(t, content, "GetMyschemaUser", "method name should use schema-prefixed name")
}

// TestGenerate_SameNameTablesInDifferentSchemas verifies that two tables
// with the same bare name in different schemas produce distinct output files,
// method names, and SQL with correct schema qualification.
func TestGenerate_SameNameTablesInDifferentSchemas(t *testing.T) {
	tmp := t.TempDir()
	migrationsDir := filepath.Join(tmp, "sql", "migrations")
	require.NoError(t, os.MkdirAll(migrationsDir, 0o755))

	schemaSQL := `CREATE SCHEMA shop;
CREATE TABLE orders (
  id TEXT PRIMARY KEY,
  total TEXT NOT NULL
);
CREATE TABLE shop.orders (
  id TEXT PRIMARY KEY,
  amount TEXT NOT NULL
);`
	require.NoError(t, os.WriteFile(filepath.Join(migrationsDir, "001_init.sql"), []byte(schemaSQL), 0o644))

	configPath := filepath.Join(tmp, "pgxgen.yaml")
	cfg := &config.V2Config{
		Version: "2",
		Schemas: []config.SchemaConfig{
			{
				Name:      "main",
				Engine:    "postgresql",
				SchemaDir: "sql/migrations",
				Defaults: &config.DefaultsConfig{
					QueriesDirPrefix: "sql/queries",
					OutputDirPrefix:  "internal/store/repos",
				},
				Tables: map[string]config.TableConfig{
					"orders": {
						PrimaryColumn: "id",
						Crud: &config.TableCrudConfig{
							Methods: map[string]*config.MethodConfig{
								"get":    {},
								"create": {Returning: "*"},
							},
						},
					},
					"shop.orders": {
						PrimaryColumn: "id",
						Crud: &config.TableCrudConfig{
							Methods: map[string]*config.MethodConfig{
								"get":    {},
								"create": {Returning: "*"},
							},
						},
					},
				},
			},
		},
	}

	orch := NewOrchestrator(logger.New(), cfg, configPath)
	_, err := orch.Generate(context.Background(), GenerateOpts{Targets: []string{"crud"}})
	require.NoError(t, err)

	// Public schema table — bare name paths
	publicPath := filepath.Join(tmp, "sql", "queries", "orders", "orders_gen.sql")
	assert.FileExists(t, publicPath)

	// Shop schema table — schema-prefixed paths
	shopPath := filepath.Join(tmp, "sql", "queries", "shop_orders", "shop_orders_gen.sql")
	assert.FileExists(t, shopPath)

	// Verify public schema SQL
	publicData, err := os.ReadFile(publicPath)
	require.NoError(t, err)
	publicSQL := string(publicData)
	assert.Contains(t, publicSQL, "SELECT * FROM orders")
	assert.Contains(t, publicSQL, "INSERT INTO orders")
	assert.Contains(t, publicSQL, "-- name: CreateOrder :one")
	assert.Contains(t, publicSQL, "-- name: GetOrder :one")

	// Verify shop schema SQL
	shopData, err := os.ReadFile(shopPath)
	require.NoError(t, err)
	shopSQL := string(shopData)
	assert.Contains(t, shopSQL, "SELECT * FROM shop.orders")
	assert.Contains(t, shopSQL, "INSERT INTO shop.orders")
	assert.Contains(t, shopSQL, "-- name: CreateShopOrder :one")
	assert.Contains(t, shopSQL, "-- name: GetShopOrder :one")
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
