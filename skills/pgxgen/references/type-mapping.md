# SQL to Go Type Mapping

pgxgen maps SQL column types to Go types based on the database engine, nullability, and configured `sql_package`.

## Type Resolution Priority

When resolving a column's Go type, pgxgen checks in this order (highest priority first):

1. **sqlc column overrides** — per-column `go_type` in `sqlc.overrides.columns`
2. **sqlc type overrides** — `db_type` → `go_type` in `sqlc.overrides.types`
3. **models type_overrides** — `sql_type` → `go_type` in `models.type_overrides`
4. **Default typemap** — engine-specific built-in mapping (tables below)
5. **Pointer replacement** — when `models.emit_pointers_for_null` is set, the resolved nullable wrapper type is replaced with a `*T` pointer (see the "Nullable (pointers)" columns below)

## PostgreSQL Type Mapping

### Integers

| SQL Type                   | NOT NULL | Nullable (pgx/v5) | Nullable (database/sql) | Nullable (pointers) |
| -------------------------- | -------- | ----------------- | ----------------------- | ------------------- |
| smallint, int2, serial2    | `int16`  | `pgtype.Int2`     | `sql.NullInt16`         | `*int16`            |
| integer, int, int4, serial | `int32`  | `pgtype.Int4`     | `sql.NullInt32`         | `*int32`            |
| bigint, int8, bigserial    | `int64`  | `pgtype.Int8`     | `sql.NullInt64`         | `*int64`            |

### Floating Point

| SQL Type                 | NOT NULL                                | Nullable (pgx/v5) | Nullable (database/sql) | Nullable (pointers) |
| ------------------------ | --------------------------------------- | ----------------- | ----------------------- | ------------------- |
| real, float4             | `float32`                               | `pgtype.Float4`   | `sql.NullFloat64`       | `*float32`          |
| double precision, float8 | `float64`                               | `pgtype.Float8`   | `sql.NullFloat64`       | `*float64`          |
| numeric, money           | `pgtype.Numeric` (pgx) / `string` (sql) | `pgtype.Numeric`  | `sql.NullString`        | `*string`           |

### Strings

| SQL Type                          | NOT NULL | Nullable (pgx/v5) | Nullable (database/sql) | Nullable (pointers) |
| --------------------------------- | -------- | ----------------- | ----------------------- | ------------------- |
| text, varchar, char, citext, name | `string` | `pgtype.Text`     | `sql.NullString`        | `*string`           |

### Boolean

| SQL Type      | NOT NULL | Nullable (pgx/v5) | Nullable (database/sql) | Nullable (pointers) |
| ------------- | -------- | ----------------- | ----------------------- | ------------------- |
| boolean, bool | `bool`   | `pgtype.Bool`     | `sql.NullBool`          | `*bool`             |

### Date/Time

| SQL Type    | NOT NULL                                    | Nullable (pgx/v5)    | Nullable (database/sql) | Nullable (pointers) |
| ----------- | ------------------------------------------- | -------------------- | ----------------------- | ------------------- |
| date        | `pgtype.Date` (pgx/v5) / `time.Time`        | `pgtype.Date`        | `sql.NullTime`          | `*time.Time`        |
| timestamp   | `pgtype.Timestamp` (pgx/v5) / `time.Time`   | `pgtype.Timestamp`   | `sql.NullTime`          | `*time.Time`        |
| timestamptz | `pgtype.Timestamptz` (pgx/v5) / `time.Time` | `pgtype.Timestamptz` | `sql.NullTime`          | `*time.Time`        |
| interval    | `pgtype.Interval` (pgx/v5) / `int64`        | `pgtype.Interval`    | `sql.NullInt64`         | `*int64`            |

### UUID

| SQL Type | NOT NULL                             | Nullable (pgx/v5) | Nullable (other) | Nullable (pointers) |
| -------- | ------------------------------------ | ----------------- | ---------------- | ------------------- |
| uuid     | `pgtype.UUID` (pgx/v5) / `uuid.UUID` | `pgtype.UUID`     | `uuid.NullUUID`  | `*uuid.UUID`        |

### JSON

| SQL Type | pgx/v5   | pgx/v4         | database/sql      |
| -------- | -------- | -------------- | ----------------- |
| json     | `[]byte` | `pgtype.JSON`  | `json.RawMessage` |
| jsonb    | `[]byte` | `pgtype.JSONB` | `json.RawMessage` |

### Binary

| SQL Type    | Go Type  |
| ----------- | -------- |
| bytea, blob | `[]byte` |

### Network

| SQL Type | pgx/v5 (NOT NULL)  | pgx/v5 (nullable)  | pgx/v4           | database/sql |
| -------- | ------------------ | ------------------ | ---------------- | ------------ |
| inet     | `netip.Addr`       | `*netip.Addr`      | `pgtype.Inet`    | `string`     |
| cidr     | `netip.Prefix`     | `*netip.Prefix`    | `pgtype.CIDR`    | `string`     |
| macaddr  | `net.HardwareAddr` | `net.HardwareAddr` | `pgtype.Macaddr` | `string`     |

### Ranges (pgx only)

| SQL Type  | pgx/v5                             | pgx/v4             |
| --------- | ---------------------------------- | ------------------ |
| daterange | `pgtype.Range[pgtype.Date]`        | `pgtype.Daterange` |
| tsrange   | `pgtype.Range[pgtype.Timestamp]`   | `pgtype.Tsrange`   |
| tstzrange | `pgtype.Range[pgtype.Timestamptz]` | `pgtype.Tstzrange` |
| numrange  | `pgtype.Range[pgtype.Numeric]`     | `pgtype.Numrange`  |
| int4range | `pgtype.Range[pgtype.Int4]`        | `pgtype.Int4range` |
| int8range | `pgtype.Range[pgtype.Int8]`        | `pgtype.Int8range` |

### Other

| SQL Type      | pgx/v5                   | pgx/v4          | database/sql                |
| ------------- | ------------------------ | --------------- | --------------------------- |
| hstore        | `pgtype.Hstore`          | `pgtype.Hstore` | `interface{}`               |
| bit, varbit   | `pgtype.Bits`            | `pgtype.Varbit` | `interface{}`               |
| ltree, lquery | `string` / `pgtype.Text` | `string`        | `string` / `sql.NullString` |

### Enums

PostgreSQL `CREATE TYPE ... AS ENUM` generates:

- NOT NULL: `EnumName` (CamelCase Go type)
- Nullable: `NullEnumName`

## MySQL Type Mapping

| SQL Type                                            | NOT NULL                      | Nullable          |
| --------------------------------------------------- | ----------------------------- | ----------------- |
| varchar, text, char, tinytext, mediumtext, longtext | `string`                      | `sql.NullString`  |
| tinyint(1)                                          | `bool`                        | `sql.NullBool`    |
| tinyint                                             | `int8` / `uint8` (unsigned)   | `sql.NullInt16`   |
| smallint                                            | `int16` / `uint16` (unsigned) | `sql.NullInt16`   |
| int, integer, mediumint                             | `int32` / `uint32` (unsigned) | `sql.NullInt32`   |
| bigint                                              | `int64` / `uint64` (unsigned) | `sql.NullInt64`   |
| year                                                | `int16`                       | `sql.NullInt16`   |
| float, double, real                                 | `float64`                     | `sql.NullFloat64` |
| decimal, dec, fixed                                 | `string`                      | `sql.NullString`  |
| boolean, bool                                       | `bool`                        | `sql.NullBool`    |
| date, datetime, timestamp, time                     | `time.Time`                   | `sql.NullTime`    |
| json                                                | `json.RawMessage`             | `json.RawMessage` |
| blob, binary, varbinary                             | `[]byte`                      | `sql.NullString`  |
| enum                                                | `string`                      | `string`          |

MySQL supports unsigned types — `uint8`, `uint16`, `uint32`, `uint64` for unsigned integer columns.

## SQLite Type Mapping

| SQL Type                               | NOT NULL          | Nullable          | Nullable (pointers) |
| -------------------------------------- | ----------------- | ----------------- | ------------------- |
| integer, int, bigint, smallint, etc.   | `int64`           | `sql.NullInt64`   | `*int64`            |
| text, varchar, char, clob, nchar, etc. | `string`          | `sql.NullString`  | `*string`           |
| real, double, float                    | `float64`         | `sql.NullFloat64` | `*float64`          |
| boolean, bool                          | `bool`            | `sql.NullBool`    | `*bool`             |
| date, datetime, timestamp              | `time.Time`       | `sql.NullTime`    | `*time.Time`        |
| decimal, numeric                       | `float64`         | `sql.NullFloat64` | `*float64`          |
| json, jsonb                            | `json.RawMessage` | `json.RawMessage` | —                   |
| blob                                   | `[]byte`          | `[]byte`          | —                   |

## Overriding Types

### In models config

```yaml
models:
  type_overrides:
    - sql_type: uuid
      go_type: uuid.UUID
      import: github.com/google/uuid
    - sql_type: numeric
      go_type: decimal.Decimal
      import: github.com/shopspring/decimal
```

### In sqlc overrides

```yaml
sqlc:
  overrides:
    types:
      - db_type: uuid
        go_type: "github.com/google/uuid.UUID"
      - db_type: uuid
        nullable: true
        go_type: "github.com/google/uuid.NullUUID"
    columns:
      - column: orders.amount
        go_type: "github.com/shopspring/decimal.Decimal"
```
