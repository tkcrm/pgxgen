# sqlc Type Overrides Reference

## Overview

Overrides map SQL types to custom Go types. They can be defined per-package
(in `sql[].gen.go.overrides`) or globally (in top-level `overrides.go`).

Two targeting modes:

- **db_type** — applies to all columns of that SQL type
- **column** — applies to a specific column (takes precedence over db_type)

## Syntax

### String shorthand

```yaml
overrides:
  - db_type: "uuid"
    go_type: "github.com/google/uuid.UUID"
```

Format: `"import/path.TypeName"`. sqlc infers the package name from the last path segment.

### Map form (full control)

```yaml
overrides:
  - db_type: "uuid"
    go_type:
      import: "github.com/google/uuid"
      package: "uuid" # Optional if matches last import segment
      type: "UUID"
      pointer: false # Use *UUID
      slice: false # Use []UUID
```

### Column-specific override

```yaml
overrides:
  - column: "users.metadata"
    go_type:
      import: "encoding/json"
      type: "RawMessage"
```

Column format: `"table.column"` or `"schema.table.column"`.

### Nullable variants

```yaml
overrides:
  - db_type: "uuid"
    nullable: true
    go_type:
      import: "github.com/google/uuid"
      type: "UUID"
      pointer: true
```

`nullable: true` only works with `db_type`, not `column`.

### Unsigned (MySQL only)

```yaml
overrides:
  - db_type: "int"
    unsigned: true
    go_type: "uint32"
```

### Custom struct tags

```yaml
overrides:
  - column: "users.email"
    go_struct_tag: 'validate:"required,email" db:"email"'
```

`go_struct_tag` uses Go reflect-style tag format. Can be combined with `go_type`.

## Common override recipes

### UUID (PostgreSQL)

```yaml
overrides:
  - db_type: "uuid"
    go_type: "github.com/google/uuid.UUID"
  - db_type: "uuid"
    nullable: true
    go_type:
      import: "github.com/google/uuid"
      type: "UUID"
      pointer: true
```

### JSON/JSONB (PostgreSQL)

Map to a custom struct:

```yaml
overrides:
  - column: "users.preferences"
    go_type:
      import: "myapp/types"
      type: "UserPreferences"
```

Map to `json.RawMessage` (defer parsing):

```yaml
overrides:
  - db_type: "jsonb"
    go_type:
      import: "encoding/json"
      type: "RawMessage"
```

### Timestamps

Force `time.Time` for all timestamp types:

```yaml
overrides:
  - db_type: "pg_catalog.timestamptz"
    go_type:
      import: "time"
      type: "Time"
  - db_type: "pg_catalog.timestamptz"
    nullable: true
    go_type:
      import: "time"
      type: "Time"
      pointer: true
```

### PostgreSQL arrays

```yaml
overrides:
  - db_type: "text[]"
    go_type:
      import: "github.com/lib/pq"
      type: "StringArray"
  - db_type: "integer[]"
    go_type:
      import: "github.com/lib/pq"
      type: "Int64Array"
```

With pgx/v5, arrays are natively supported — no override needed.

### Numeric/decimal

```yaml
overrides:
  - db_type: "numeric"
    go_type:
      import: "github.com/shopspring/decimal"
      type: "Decimal"
  - db_type: "numeric"
    nullable: true
    go_type:
      import: "github.com/shopspring/decimal"
      type: "NullDecimal"
```

### inet/cidr (PostgreSQL)

```yaml
overrides:
  - db_type: "inet"
    go_type:
      import: "net/netip"
      type: "Addr"
  - db_type: "cidr"
    go_type:
      import: "net/netip"
      type: "Prefix"
```

### Enums as custom type

By default sqlc generates string-aliased enum types. To use a custom type:

```yaml
overrides:
  - column: "users.role"
    go_type:
      import: "myapp/types"
      type: "Role"
```

### Standard library null types (database/sql)

```yaml
overrides:
  - db_type: "text"
    nullable: true
    go_type: "database/sql.NullString"
  - db_type: "integer"
    nullable: true
    go_type: "database/sql.NullInt64"
  - db_type: "boolean"
    nullable: true
    go_type: "database/sql.NullBool"
```

## Global vs per-package overrides

**Per-package** (in `sql[].gen.go.overrides`):

```yaml
sql:
  - gen:
      go:
        overrides:
          - db_type: "uuid"
            go_type: "github.com/google/uuid.UUID"
```

**Global** (applies to all packages):

```yaml
overrides:
  go:
    overrides:
      - db_type: "uuid"
        go_type: "github.com/google/uuid.UUID"
```

Global overrides also support an `engine` filter:

```yaml
overrides:
  go:
    overrides:
      - db_type: "uuid"
        engine: "postgresql"
        go_type: "github.com/google/uuid.UUID"
```

## Priority

1. Column-specific overrides (highest priority)
2. Per-package db_type overrides
3. Global db_type overrides (lowest priority)
