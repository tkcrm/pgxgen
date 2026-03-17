package typemap

import (
	"strings"

	"github.com/tkcrm/pgxgen/internal/sqlparser/catalog"
)

type mysqlMapper struct{}

func (m *mysqlMapper) GoType(col *catalog.Column, enums []*catalog.Enum, opts Options) string {
	columnType := strings.ToLower(col.Type)
	notNull := col.NotNull || col.IsArray
	unsigned := col.IsUnsigned

	switch columnType {
	case "varchar", "text", "char", "tinytext", "mediumtext", "longtext":
		if notNull {
			return "string"
		}
		return "sql.NullString"

	case "tinyint":
		if col.Length != nil && *col.Length == 1 {
			if notNull {
				return "bool"
			}
			return "sql.NullBool"
		}
		if notNull {
			if unsigned {
				return "uint8"
			}
			return "int8"
		}
		return "sql.NullInt16"

	case "year":
		if notNull {
			return "int16"
		}
		return "sql.NullInt16"

	case "smallint":
		if notNull {
			if unsigned {
				return "uint16"
			}
			return "int16"
		}
		return "sql.NullInt16"

	case "int", "integer", "mediumint":
		if notNull {
			if unsigned {
				return "uint32"
			}
			return "int32"
		}
		return "sql.NullInt32"

	case "bigint":
		if notNull {
			if unsigned {
				return "uint64"
			}
			return "int64"
		}
		return "sql.NullInt64"

	case "blob", "binary", "varbinary", "tinyblob", "mediumblob", "longblob":
		if notNull {
			return "[]byte"
		}
		return "sql.NullString"

	case "double", "double precision", "real", "float":
		if notNull {
			return "float64"
		}
		return "sql.NullFloat64"

	case "decimal", "dec", "fixed":
		if notNull {
			return "string"
		}
		return "sql.NullString"

	case "enum":
		return "string"

	case "date", "timestamp", "datetime", "time":
		if notNull {
			return "time.Time"
		}
		return "sql.NullTime"

	case "boolean", "bool":
		if notNull {
			return "bool"
		}
		return "sql.NullBool"

	case "json":
		return "json.RawMessage"

	case "any":
		return "interface{}"

	default:
		// Check for enum types
		for _, enum := range enums {
			if strings.EqualFold(enum.Name, columnType) {
				if notNull {
					return structName(enum.Name)
				}
				return "Null" + structName(enum.Name)
			}
		}
		return "interface{}"
	}
}
