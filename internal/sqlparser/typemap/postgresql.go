package typemap

import (
	"strings"

	"github.com/tkcrm/pgxgen/internal/sqlparser/catalog"
)

type postgresMapper struct{}

func (m *postgresMapper) GoType(col *catalog.Column, enums []*catalog.Enum, opts Options) string {
	columnType := strings.ToLower(col.Type)
	notNull := col.NotNull || col.IsArray
	driver := parseDriver(opts.SqlPackage)
	emitPointersForNull := (driver == driverPGXV4 || driver == driverPGXV5) && opts.EmitPointersForNull

	switch columnType {
	case "serial", "serial4", "pg_catalog.serial4":
		if notNull {
			return "int32"
		}
		if emitPointersForNull {
			return "*int32"
		}
		if driver == driverPGXV5 {
			return "pgtype.Int4"
		}
		return "sql.NullInt32"

	case "bigserial", "serial8", "pg_catalog.serial8":
		if notNull {
			return "int64"
		}
		if emitPointersForNull {
			return "*int64"
		}
		if driver == driverPGXV5 {
			return "pgtype.Int8"
		}
		return "sql.NullInt64"

	case "smallserial", "serial2", "pg_catalog.serial2":
		if notNull {
			return "int16"
		}
		if emitPointersForNull {
			return "*int16"
		}
		if driver == driverPGXV5 {
			return "pgtype.Int2"
		}
		return "sql.NullInt16"

	case "integer", "int", "int4", "pg_catalog.int4":
		if notNull {
			return "int32"
		}
		if emitPointersForNull {
			return "*int32"
		}
		if driver == driverPGXV5 {
			return "pgtype.Int4"
		}
		return "sql.NullInt32"

	case "bigint", "int8", "pg_catalog.int8":
		if notNull {
			return "int64"
		}
		if emitPointersForNull {
			return "*int64"
		}
		if driver == driverPGXV5 {
			return "pgtype.Int8"
		}
		return "sql.NullInt64"

	case "smallint", "int2", "pg_catalog.int2":
		if notNull {
			return "int16"
		}
		if emitPointersForNull {
			return "*int16"
		}
		if driver == driverPGXV5 {
			return "pgtype.Int2"
		}
		return "sql.NullInt16"

	case "float", "double precision", "float8", "pg_catalog.float8":
		if notNull {
			return "float64"
		}
		if emitPointersForNull {
			return "*float64"
		}
		if driver == driverPGXV5 {
			return "pgtype.Float8"
		}
		return "sql.NullFloat64"

	case "real", "float4", "pg_catalog.float4":
		if notNull {
			return "float32"
		}
		if emitPointersForNull {
			return "*float32"
		}
		if driver == driverPGXV5 {
			return "pgtype.Float4"
		}
		return "sql.NullFloat64"

	case "numeric", "pg_catalog.numeric", "money":
		if driver == driverPGXV4 || driver == driverPGXV5 {
			return "pgtype.Numeric"
		}
		if notNull {
			return "string"
		}
		if emitPointersForNull {
			return "*string"
		}
		return "sql.NullString"

	case "boolean", "bool", "pg_catalog.bool":
		if notNull {
			return "bool"
		}
		if emitPointersForNull {
			return "*bool"
		}
		if driver == driverPGXV5 {
			return "pgtype.Bool"
		}
		return "sql.NullBool"

	case "json", "pg_catalog.json":
		switch driver {
		case driverPGXV5:
			return "[]byte"
		case driverPGXV4:
			return "pgtype.JSON"
		default:
			if notNull {
				return "json.RawMessage"
			}
			return "json.RawMessage"
		}

	case "jsonb", "pg_catalog.jsonb":
		switch driver {
		case driverPGXV5:
			return "[]byte"
		case driverPGXV4:
			return "pgtype.JSONB"
		default:
			if notNull {
				return "json.RawMessage"
			}
			return "json.RawMessage"
		}

	case "bytea", "blob", "pg_catalog.bytea":
		return "[]byte"

	case "date":
		if driver == driverPGXV5 {
			return "pgtype.Date"
		}
		if notNull {
			return "time.Time"
		}
		if emitPointersForNull {
			return "*time.Time"
		}
		return "sql.NullTime"

	case "pg_catalog.time":
		if driver == driverPGXV5 {
			return "pgtype.Time"
		}
		if notNull {
			return "time.Time"
		}
		if emitPointersForNull {
			return "*time.Time"
		}
		return "sql.NullTime"

	case "pg_catalog.timetz":
		if notNull {
			return "time.Time"
		}
		if emitPointersForNull {
			return "*time.Time"
		}
		return "sql.NullTime"

	case "pg_catalog.timestamp", "timestamp":
		if driver == driverPGXV5 {
			return "pgtype.Timestamp"
		}
		if notNull {
			return "time.Time"
		}
		if emitPointersForNull {
			return "*time.Time"
		}
		return "sql.NullTime"

	case "pg_catalog.timestamptz", "timestamptz":
		if driver == driverPGXV5 {
			return "pgtype.Timestamptz"
		}
		if notNull {
			return "time.Time"
		}
		if emitPointersForNull {
			return "*time.Time"
		}
		return "sql.NullTime"

	case "text", "pg_catalog.varchar", "pg_catalog.bpchar", "string", "citext", "name",
		"varchar", "character varying", "bpchar", "char", "character":
		if notNull {
			return "string"
		}
		if emitPointersForNull {
			return "*string"
		}
		if driver == driverPGXV5 {
			return "pgtype.Text"
		}
		return "sql.NullString"

	case "uuid":
		if driver == driverPGXV5 {
			return "pgtype.UUID"
		}
		if notNull {
			return "uuid.UUID"
		}
		if emitPointersForNull {
			return "*uuid.UUID"
		}
		return "uuid.NullUUID"

	case "inet":
		switch driver {
		case driverPGXV5:
			if notNull {
				return "netip.Addr"
			}
			return "*netip.Addr"
		case driverPGXV4:
			return "pgtype.Inet"
		default:
			return "string"
		}

	case "cidr":
		switch driver {
		case driverPGXV5:
			if notNull {
				return "netip.Prefix"
			}
			return "*netip.Prefix"
		case driverPGXV4:
			return "pgtype.CIDR"
		default:
			return "string"
		}

	case "macaddr", "macaddr8":
		switch driver {
		case driverPGXV5:
			return "net.HardwareAddr"
		case driverPGXV4:
			return "pgtype.Macaddr"
		default:
			return "string"
		}

	case "ltree", "lquery", "ltxtquery":
		if notNull {
			return "string"
		}
		if emitPointersForNull {
			return "*string"
		}
		if driver == driverPGXV5 {
			return "pgtype.Text"
		}
		return "sql.NullString"

	case "interval", "pg_catalog.interval":
		if driver == driverPGXV5 {
			return "pgtype.Interval"
		}
		if notNull {
			return "int64"
		}
		if emitPointersForNull {
			return "*int64"
		}
		return "sql.NullInt64"

	case "daterange":
		if driver == driverPGXV5 {
			return "pgtype.Range[pgtype.Date]"
		}
		if driver == driverPGXV4 {
			return "pgtype.Daterange"
		}
		return "interface{}"

	case "tsrange":
		if driver == driverPGXV5 {
			return "pgtype.Range[pgtype.Timestamp]"
		}
		if driver == driverPGXV4 {
			return "pgtype.Tsrange"
		}
		return "interface{}"

	case "tstzrange":
		if driver == driverPGXV5 {
			return "pgtype.Range[pgtype.Timestamptz]"
		}
		if driver == driverPGXV4 {
			return "pgtype.Tstzrange"
		}
		return "interface{}"

	case "numrange":
		if driver == driverPGXV5 {
			return "pgtype.Range[pgtype.Numeric]"
		}
		if driver == driverPGXV4 {
			return "pgtype.Numrange"
		}
		return "interface{}"

	case "int4range":
		if driver == driverPGXV5 {
			return "pgtype.Range[pgtype.Int4]"
		}
		if driver == driverPGXV4 {
			return "pgtype.Int4range"
		}
		return "interface{}"

	case "int8range":
		if driver == driverPGXV5 {
			return "pgtype.Range[pgtype.Int8]"
		}
		if driver == driverPGXV4 {
			return "pgtype.Int8range"
		}
		return "interface{}"

	case "hstore":
		if driver == driverPGXV4 || driver == driverPGXV5 {
			return "pgtype.Hstore"
		}
		return "interface{}"

	case "bit", "varbit", "pg_catalog.bit", "pg_catalog.varbit":
		if driver == driverPGXV5 {
			return "pgtype.Bits"
		}
		if driver == driverPGXV4 {
			return "pgtype.Varbit"
		}
		return "interface{}"

	case "void", "any":
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
