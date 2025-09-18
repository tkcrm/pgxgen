package crud

import (
	"strings"

	"github.com/tkcrm/pgxgen/internal/config"
)

const (
	METHOD_ALL    config.MethodType = "*"
	METHOD_CREATE config.MethodType = "create"
	METHOD_UPDATE config.MethodType = "update"
	METHOD_DELETE config.MethodType = "delete"
	METHOD_GET    config.MethodType = "get"
	METHOD_FIND   config.MethodType = "find"
	METHOD_TOTAL  config.MethodType = "total"
	METHOD_EXISTS config.MethodType = "exists"
)

type tables map[string]*tableMetaData

type tableMetaData struct {
	columns []string
}

func (t tables) getTableMetaData(tableName string) *tableMetaData {
	for name, metaData := range t {
		if name == tableName {
			return metaData
		}
	}
	return nil
}

type engineType string

const (
	EngineTypePostgres engineType = "postgresql"
	EngineTypeMysql    engineType = "mysql"
	EngineTypeSqlite   engineType = "sqlite"
)

func (e engineType) String() string {
	return string(e)
}

func (s engineType) Valid() bool {
	switch s {
	case EngineTypePostgres, EngineTypeMysql, EngineTypeSqlite:
		return true
	}
	return false
}

type processParams struct {
	builder      *strings.Builder
	table        string
	metaData     tableMetaData
	methodParams config.Method
	tableParams  config.TableParams
	engine       engineType
}
