package sqlc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tkcrm/pgxgen/internal/config"
)

func TestBuildSqlcConfig_SameNamedTablesInDifferentSchemas(t *testing.T) {
	schema := &config.SchemaConfig{
		Engine:    "postgresql",
		SchemaDir: "sql/migrations",
		Defaults: &config.DefaultsConfig{
			QueriesDirPrefix: "sql/queries",
			OutputDirPrefix:  "internal/store/repos",
		},
		Tables: map[string]config.TableConfig{
			"orders": {
				PrimaryColumn: "id",
			},
			"shop.orders": {
				PrimaryColumn: "id",
			},
		},
	}

	cfg := BuildSqlcConfig(schema)
	require.Len(t, cfg.SQL, 2)

	// Find entries by queries path (sorted alphabetically: "orders" before "shop.orders")
	var publicEntry, shopEntry sqlcSQLEntry
	for _, entry := range cfg.SQL {
		if entry.Queries == "../sql/queries/orders" {
			publicEntry = entry
		}
		if entry.Queries == "../sql/queries/shop_orders" {
			shopEntry = entry
		}
	}

	// Public schema table — bare name directories
	assert.Equal(t, "../sql/queries/orders", publicEntry.Queries)
	assert.Equal(t, "../internal/store/repos/orders", publicEntry.Gen.Go.Out)

	// Shop schema table — schema-prefixed directories
	assert.Equal(t, "../sql/queries/shop_orders", shopEntry.Queries)
	assert.Equal(t, "../internal/store/repos/shop_orders", shopEntry.Gen.Go.Out)

	// Both use the same schema dir
	assert.Equal(t, "../sql/migrations", publicEntry.Schema)
	assert.Equal(t, "../sql/migrations", shopEntry.Schema)
}

func TestBuildSqlcConfig_PlainTableUnchanged(t *testing.T) {
	schema := &config.SchemaConfig{
		Engine:    "postgresql",
		SchemaDir: "sql/migrations",
		Defaults: &config.DefaultsConfig{
			QueriesDirPrefix: "sql/queries",
			OutputDirPrefix:  "internal/store/repos",
		},
		Tables: map[string]config.TableConfig{
			"users": {
				PrimaryColumn: "id",
			},
		},
	}

	cfg := BuildSqlcConfig(schema)
	require.Len(t, cfg.SQL, 1)

	entry := cfg.SQL[0]
	assert.Equal(t, "../sql/queries/users", entry.Queries)
	assert.Equal(t, "../internal/store/repos/users", entry.Gen.Go.Out)
}
