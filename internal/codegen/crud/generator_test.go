package crud

import (
	"os"
	"path/filepath"
	"testing"

	sqlcpkg "github.com/sqlc-dev/sqlc/pkg/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/engine"
	"gopkg.in/yaml.v3"
)

func newTestGenerator(t *testing.T, engineName string) *Generator {
	t.Helper()
	eng, err := engine.New(engineName)
	require.NoError(t, err)
	return New(eng)
}

func generateTable(t *testing.T, engineName, tableName string, tableConfig config.TableConfig, defaultCrud *config.CrudDefaultsConfig, columns []string) string {
	t.Helper()
	gen := newTestGenerator(t, engineName)
	result, err := gen.GenerateTable(tableName, tableConfig, defaultCrud, columns)
	require.NoError(t, err)
	return string(result)
}

var testColumns = []string{"id", "name", "email", "created_at", "updated_at"}

func TestBatchCreate_Basic(t *testing.T) {
	tests := []struct {
		name     string
		engine   string
		contains []string
	}{
		{
			name:   "postgresql",
			engine: "postgresql",
			contains: []string{
				"-- name: BatchCreateUser :copyfrom",
				"INSERT INTO users (id, name, email, created_at, updated_at)",
				"VALUES ($1, $2, $3, $4, $5);",
			},
		},
		{
			name:   "mysql",
			engine: "mysql",
			contains: []string{
				"-- name: BatchCreateUser :copyfrom",
				"INSERT INTO users (id, name, email, created_at, updated_at)",
				"VALUES (?, ?, ?, ?, ?);",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tableConfig := config.TableConfig{
				Crud: &config.TableCrudConfig{
					Methods: map[string]*config.MethodConfig{
						"batch_create": {},
					},
				},
			}

			result := generateTable(t, tt.engine, "users", tableConfig, nil, testColumns)

			for _, expected := range tt.contains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestBatchCreate_UnsupportedSQLite(t *testing.T) {
	tableConfig := config.TableConfig{
		Crud: &config.TableCrudConfig{
			Methods: map[string]*config.MethodConfig{
				"batch_create": {},
			},
		},
	}

	gen := newTestGenerator(t, "sqlite")
	_, err := gen.GenerateTable("users", tableConfig, nil, testColumns)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "batch_create is not supported for SQLite")
}

func TestBatchCreate_SkipColumns(t *testing.T) {
	tableConfig := config.TableConfig{
		Crud: &config.TableCrudConfig{
			Methods: map[string]*config.MethodConfig{
				"batch_create": {
					SkipColumns: []string{"id", "created_at", "updated_at"},
				},
			},
		},
	}

	result := generateTable(t, "postgresql", "users", tableConfig, nil, testColumns)

	assert.Contains(t, result, "-- name: BatchCreateUser :copyfrom")
	assert.Contains(t, result, "INSERT INTO users (name, email)")
	assert.Contains(t, result, "VALUES ($1, $2);")
	assert.NotContains(t, result, "id")
	assert.NotContains(t, result, "created_at")
	assert.NotContains(t, result, "updated_at")
}

func TestBatchCreate_ColumnValues(t *testing.T) {
	tableConfig := config.TableConfig{
		Crud: &config.TableCrudConfig{
			Methods: map[string]*config.MethodConfig{
				"batch_create": {
					SkipColumns: []string{"id"},
					ColumnValues: map[string]string{
						"created_at": "now()",
					},
				},
			},
		},
	}

	result := generateTable(t, "postgresql", "users", tableConfig, nil, testColumns)

	assert.Contains(t, result, "-- name: BatchCreateUser :copyfrom")
	// created_at should be excluded because column_values can't work with copyfrom
	assert.NotContains(t, result, "created_at")
	assert.NotContains(t, result, "now()")
	assert.Contains(t, result, "INSERT INTO users (name, email, updated_at)")
	assert.Contains(t, result, "VALUES ($1, $2, $3);")
}

func TestBatchCreate_CustomName(t *testing.T) {
	tableConfig := config.TableConfig{
		Crud: &config.TableCrudConfig{
			Methods: map[string]*config.MethodConfig{
				"batch_create": {
					Name: "InsertBulkUsers",
				},
			},
		},
	}

	result := generateTable(t, "postgresql", "users", tableConfig, nil, testColumns)

	assert.Contains(t, result, "-- name: InsertBulkUsers :copyfrom")
}

func TestBatchCreate_ExcludeTableName(t *testing.T) {
	tableConfig := config.TableConfig{
		Crud: &config.TableCrudConfig{
			Methods: map[string]*config.MethodConfig{
				"batch_create": {},
			},
		},
	}

	defaultCrud := &config.CrudDefaultsConfig{
		ExcludeTableName: true,
	}

	result := generateTable(t, "postgresql", "users", tableConfig, defaultCrud, testColumns)

	assert.Contains(t, result, "-- name: BatchCreate :copyfrom")
}

// TestBatchCreate_SqlcIntegration verifies that sqlc can process the generated batch_create SQL.
// Note: :copyfrom is only supported by pgx (PostgreSQL) and go-sql-driver/mysql.
// SQLite does not support :copyfrom in sqlc.
// MySQL :copyfrom does not support TIMESTAMP columns.
func TestBatchCreate_SqlcIntegration(t *testing.T) {
	tests := []struct {
		name       string
		engine     string
		schema     string
		columns    []string
		skipCols   []string
		sqlPackage string
		sqlDriver  string
	}{
		{
			name:   "postgresql",
			engine: "postgresql",
			schema: `CREATE TABLE users (
	id UUID NOT NULL PRIMARY KEY,
	name VARCHAR(255) NOT NULL,
	email VARCHAR(255) NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT now(),
	updated_at TIMESTAMP
);`,
			columns:    []string{"id", "name", "email", "created_at", "updated_at"},
			skipCols:   []string{"id"},
			sqlPackage: "pgx/v5",
		},
		{
			name:   "mysql",
			engine: "mysql",
			// MySQL copyfrom does not support TIMESTAMP columns
			schema: `CREATE TABLE users (
	id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
	name VARCHAR(255) NOT NULL,
	email VARCHAR(255) NOT NULL,
	is_active BOOLEAN NOT NULL DEFAULT FALSE
);`,
			columns:   []string{"id", "name", "email", "is_active"},
			skipCols:  []string{"id"},
			sqlDriver: "github.com/go-sql-driver/mysql",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create directory structure
			schemaDir := filepath.Join(tmpDir, "schema")
			queriesDir := filepath.Join(tmpDir, "queries")
			outputDir := filepath.Join(tmpDir, "output")
			require.NoError(t, os.MkdirAll(schemaDir, 0o755))
			require.NoError(t, os.MkdirAll(queriesDir, 0o755))
			require.NoError(t, os.MkdirAll(outputDir, 0o755))

			// Write schema
			require.NoError(t, os.WriteFile(
				filepath.Join(schemaDir, "schema.sql"),
				[]byte(tt.schema),
				0o644,
			))

			// Generate batch_create SQL
			tableConfig := config.TableConfig{
				Crud: &config.TableCrudConfig{
					Methods: map[string]*config.MethodConfig{
						"batch_create": {
							SkipColumns: tt.skipCols,
						},
					},
				},
			}

			sql := generateTable(t, tt.engine, "users", tableConfig, nil, tt.columns)
			require.NotEmpty(t, sql)

			// Write generated SQL
			require.NoError(t, os.WriteFile(
				filepath.Join(queriesDir, "users_gen.sql"),
				[]byte(sql),
				0o644,
			))

			// Build sqlc.yaml
			type sqlcGenGo struct {
				Out        string `yaml:"out"`
				Package    string `yaml:"package"`
				SqlPackage string `yaml:"sql_package,omitempty"`
				SqlDriver  string `yaml:"sql_driver,omitempty"`
			}
			type sqlcGen struct {
				Go sqlcGenGo `yaml:"go"`
			}
			type sqlcSQL struct {
				Schema  string  `yaml:"schema"`
				Queries string  `yaml:"queries"`
				Engine  string  `yaml:"engine"`
				Gen     sqlcGen `yaml:"gen"`
			}
			type sqlcConfig struct {
				Version string    `yaml:"version"`
				SQL     []sqlcSQL `yaml:"sql"`
			}

			cfg := sqlcConfig{
				Version: "2",
				SQL: []sqlcSQL{
					{
						Schema:  "schema",
						Queries: "queries",
						Engine:  tt.engine,
						Gen: sqlcGen{
							Go: sqlcGenGo{
								Out:        "output",
								Package:    "output",
								SqlPackage: tt.sqlPackage,
								SqlDriver:  tt.sqlDriver,
							},
						},
					},
				},
			}

			cfgData, err := yaml.Marshal(cfg)
			require.NoError(t, err)

			sqlcPath := filepath.Join(tmpDir, "sqlc.yaml")
			require.NoError(t, os.WriteFile(sqlcPath, cfgData, 0o644))

			// Run sqlc
			exitCode := sqlcpkg.Run([]string{"generate", "-f", sqlcPath})
			assert.Equal(t, 0, exitCode, "sqlc generate should succeed for engine %s", tt.engine)

			// Verify output files were generated
			entries, err := os.ReadDir(outputDir)
			require.NoError(t, err)
			assert.NotEmpty(t, entries, "sqlc should generate output files")
		})
	}
}
