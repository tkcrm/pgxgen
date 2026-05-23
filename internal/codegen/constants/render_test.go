package constants

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderConstants_SameNamedTablesInDifferentSchemas(t *testing.T) {
	params := ConstantsParams{
		Package: "store",
		Tables: []ConstantsTableNamesParamsItem{
			{NamePreffix: "Orders", Name: "orders", BareTableName: "orders"},
			{NamePreffix: "ShopOrders", Name: "shop.orders", BareTableName: "shop_orders"},
		},
		ColumnNames: []ConstantsColumnNamesParamsItem{
			{TableName: "orders", NamePreffix: "OrdersId", Name: "id"},
			{TableName: "orders", NamePreffix: "OrdersTotal", Name: "total"},
			{TableName: "shop.orders", NamePreffix: "ShopOrdersId", Name: "id"},
			{TableName: "shop.orders", NamePreffix: "ShopOrdersAmount", Name: "amount"},
		},
	}

	var buf bytes.Buffer
	err := RenderConstants(params, &buf)
	require.NoError(t, err)
	output := buf.String()

	// Distinct table name constants
	assert.Contains(t, output, `TableNameOrders TableName = "orders"`)
	assert.Contains(t, output, `TableNameShopOrders TableName = "shop.orders"`)

	// Distinct column name constants — no collision
	assert.Contains(t, output, `ColumnNameOrdersId ColumnName = "id"`)
	assert.Contains(t, output, `ColumnNameOrdersTotal ColumnName = "total"`)
	assert.Contains(t, output, `ColumnNameShopOrdersId ColumnName = "id"`)
	assert.Contains(t, output, `ColumnNameShopOrdersAmount ColumnName = "amount"`)

	// Distinct ColumnNames() functions
	assert.Contains(t, output, "func OrdersColumnNames() ColumnNames {")
	assert.Contains(t, output, "func ShopOrdersColumnNames() ColumnNames {")

	// Each function returns only its own columns
	// OrdersColumnNames should contain OrdersId, OrdersTotal
	// ShopOrdersColumnNames should contain ShopOrdersId, ShopOrdersAmount
	assert.Contains(t, output, "ColumnNameOrdersId,\nColumnNameOrdersTotal,")
	assert.Contains(t, output, "ColumnNameShopOrdersId,\nColumnNameShopOrdersAmount,")
}

func TestRenderConstants_SingleSchemaUnchanged(t *testing.T) {
	params := ConstantsParams{
		Package: "store",
		Tables: []ConstantsTableNamesParamsItem{
			{NamePreffix: "Users", Name: "users", BareTableName: "users"},
		},
		ColumnNames: []ConstantsColumnNamesParamsItem{
			{TableName: "users", NamePreffix: "UsersId", Name: "id"},
			{TableName: "users", NamePreffix: "UsersName", Name: "name"},
		},
	}

	var buf bytes.Buffer
	err := RenderConstants(params, &buf)
	require.NoError(t, err)
	output := buf.String()

	// Plain table — no schema prefix
	assert.Contains(t, output, `TableNameUsers TableName = "users"`)
	assert.Contains(t, output, `ColumnNameUsersId ColumnName = "id"`)
	assert.Contains(t, output, "func UsersColumnNames() ColumnNames {")
}
