package golang

import (
	"log"

	"github.com/tkcrm/pgxgen/pkg/sqlc/codegen/golang/opts"
	"github.com/tkcrm/pgxgen/pkg/sqlc/codegen/sdk"
	"github.com/tkcrm/pgxgen/pkg/sqlc/debug"
	"github.com/tkcrm/pgxgen/pkg/sqlc/plugin"
)

func mysqlType(req *plugin.GenerateRequest, options *opts.Options, col *plugin.Column) string {
	columnType := sdk.DataType(col.Type)
	notNull := col.NotNull || col.IsArray
	unsigned := col.Unsigned

	switch columnType {

	case "varchar", "text", "char", "tinytext", "mediumtext", "longtext":
		if notNull {
			return "string"
		}
		return "sql.NullString"

	case "tinyint":
		if col.Length == 1 {
			if notNull {
				return "bool"
			}
			return "sql.NullBool"
		} else {
			if notNull {
				if unsigned {
					return "uint32"
				}
				return "int32"
			}
			return "sql.NullInt32"
		}

	case "int", "integer", "smallint", "mediumint", "year":
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
		// TODO: Proper Enum support
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
		for _, schema := range req.Catalog.Schemas {
			for _, enum := range schema.Enums {
				if enum.Name == columnType {
					if notNull {
						if schema.Name == req.Catalog.DefaultSchema {
							return StructName(enum.Name, options)
						}
						return StructName(schema.Name+"_"+enum.Name, options)
					} else {
						if schema.Name == req.Catalog.DefaultSchema {
							return "Null" + StructName(enum.Name, options)
						}
						return "Null" + StructName(schema.Name+"_"+enum.Name, options)
					}
				}
			}
		}
		if debug.Active {
			log.Printf("Unknown MySQL type: %s\n", columnType)
		}
		return "interface{}"

	}
}
