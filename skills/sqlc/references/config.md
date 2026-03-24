# sqlc Configuration Reference (v2)

## Table of Contents

- [Minimal config](#minimal-config)
- [Top-level fields](#top-level-fields)
- [cloud](#cloud)
- [sql](#sql)
- [sql[].database](#sqldatabase)
- [sql[].analyzer](#sqlanalyzer)
- [sql[].gen.go](#sqlgengo)
- [sql[].gen.go.overrides](#sqlgengooverrides)
- [sql[].gen.go.rename](#sqlgengorename)
- [sql[].codegen](#sqlcodegen)
- [sql[].rules](#sqlrules)
- [plugins](#plugins)
- [rules](#rules)
- [overrides (global)](#overrides-global)

## Minimal config

```yaml
version: "2"
sql:
  - schema: "schema.sql"
    queries: "query.sql"
    engine: "postgresql"
    gen:
      go:
        package: "db"
        out: "internal/db"
```

## Top-level fields

```yaml
version: "2" # Required. Always "2".
cloud: {} # Optional. sqlc Cloud settings.
sql: [] # Required. List of SQL source + codegen definitions.
plugins: [] # Optional. Plugin definitions (WASM or process).
rules: [] # Optional. Vet rule definitions.
overrides: # Optional. Global type overrides across all packages.
  go:
    rename: {}
    overrides: []
```

## cloud

```yaml
cloud:
  project: "project-id" # sqlc Cloud project ID
  organization: "org-name" # Organization name
  hostname: "custom-host" # Custom hostname (optional)
```

## sql

Each entry defines a set of SQL sources and their code generation targets.

```yaml
sql:
  - name: "identifier" # Optional. Human-readable name.
    schema: "schema.sql" # Required. File, directory, or list of paths.
    queries: "queries/" # Required. File, directory, or list of paths.
    engine: "postgresql" # Required. "postgresql", "mysql", or "sqlite".
    strict_function_checks: false # Error if SQL function not found. Default: false.
    strict_order_by: true # Error if ORDER BY is ambiguous. Default: true.
    database: {} # Optional. Database connection for analysis.
    analyzer: {} # Optional. Analyzer settings.
    gen: # Code generation targets.
      go: {} # Go code generation options.
      json: {} # JSON output options.
    codegen: [] # Plugin-based code generation.
    rules: [] # Vet rules to apply to this SQL set.
```

**schema** and **queries** accept:

- Single file: `"schema.sql"`
- Directory: `"sql/schema/"`
- List: `["schema.sql", "extensions.sql"]`

Schema supports migration files from: golang-migrate, goose, atlas, dbmate, sql-migrate, tern.
Down migrations are automatically ignored.

## sql[].database

```yaml
database:
  uri: "postgresql://user:pass@localhost:5432/db?sslmode=disable"
  managed: false # Use sqlc Cloud ephemeral database
```

- `uri` supports `${ENV_VAR}` substitution
- `managed: true` requires sqlc Cloud configuration
- Used for database-backed analysis and `sqlc/db-prepare` vet rule

## sql[].analyzer

```yaml
analyzer:
  database: true # true (default), false, or "only"
```

- `true` — Use both static and database analysis
- `false` — Skip database analysis entirely
- `"only"` — Use only database analysis

## sql[].gen.go

All Go code generation options:

```yaml
gen:
  go:
    # Required
    package: "db" # Go package name
    out: "internal/db" # Output directory

    # SQL driver
    sql_package: "pgx/v5" # "pgx/v5", "pgx/v4", "database/sql"
    sql_driver: "github.com/lib/pq" # Explicit driver import (for database/sql)

    # Struct emission
    emit_json_tags: false # Add `json:"..."` tags
    emit_db_tags: false # Add `db:"..."` tags
    emit_exact_table_names: false # Use exact table name (no singularization)
    emit_empty_slices: false # Return []T{} instead of nil for :many
    emit_result_struct_pointers: false # Return *Struct from :one queries
    emit_params_struct_pointers: false # Accept *Params in query methods
    emit_pointers_for_null_types: false # Use *T for nullable columns
    emit_enum_valid_method: false # Generate Valid() for enum types
    emit_all_enum_values: false # Generate AllXXXValues() function
    emit_sql_as_comment: false # SQL query as Go comment above method

    # Interface and queries
    emit_interface: false # Generate Querier interface
    emit_prepared_queries: false # Generate prepared statement support
    emit_exported_queries: false # Export SQL string constants
    emit_methods_with_db_argument: false # Methods accept DBTX parameter

    # Naming
    json_tags_case_style: "none" # "camel", "pascal", "snake", "none"
    json_tags_id_uppercase: false # "Id" → "ID" in JSON tags
    initialisms: ["id"] # Custom initialisms for name conversion

    # Output file names
    output_db_file_name: "db.go"
    output_models_file_name: "models.go"
    output_querier_file_name: "querier.go"
    output_batch_file_name: "batch.go"
    output_copyfrom_file_name: "copyfrom.go"
    output_files_suffix: "" # Suffix for all generated files

    # Parameters
    query_parameter_limit:
      1 # Max positional args before Params struct
      # 0 = always use Params struct

    # Build
    build_tags: "some_tag" # //go:build directive
    omit_sqlc_version: false # Omit sqlc version from header comment
    omit_unused_structs: false # Don't generate structs not used by queries
    wrap_errors: false # Wrap errors with fmt.Errorf

    # Type overrides and renaming
    overrides: [] # Per-package type overrides
    rename: {} # Name remapping

    # Inflection
    inflection_exclude_table_names: [] # Tables to exclude from singularization
```

### sql_package options

| Value          | Use case                             |
| -------------- | ------------------------------------ |
| `pgx/v5`       | PostgreSQL with pgx v5 (recommended) |
| `pgx/v4`       | PostgreSQL with pgx v4               |
| `database/sql` | Go stdlib (any database)             |

When using `database/sql` with MySQL, set `sql_driver: "github.com/go-sql-driver/mysql"`.

### query_parameter_limit

Controls when sqlc creates a Params struct vs using positional arguments:

- `query_parameter_limit: 1` (default) — Always use Params struct for 2+ params
- `query_parameter_limit: 3` — Use positional args for up to 3 params, Params struct for 4+
- `query_parameter_limit: 0` — Always use Params struct

## sql[].gen.go.overrides

Override Go type mappings for SQL types. See [overrides.md](overrides.md) for full details.

```yaml
overrides:
  - db_type: "uuid"
    go_type: "github.com/google/uuid.UUID"
  - column: "users.metadata"
    go_type:
      import: "encoding/json"
      type: "RawMessage"
```

## sql[].gen.go.rename

Rename generated Go identifiers:

```yaml
rename:
  id: "Identifier" # Column "id" → field "Identifier"
  author: "Writer" # Struct "Author" → "Writer"
  ip_address: "IPAddress" # Custom casing
```

## sql[].codegen

Plugin-based code generation (requires plugin definition in top-level `plugins`):

```yaml
codegen:
  - out: "gen/ts"
    plugin: "sqlc-gen-typescript"
    options:
      runtime: "node"
      driver: "pg"
```

## sql[].rules

List of rule names to apply to queries in this SQL set:

```yaml
rules:
  - sqlc/db-prepare
  - no-select-star
  - require-where-clause
```

Rules must be defined in top-level `rules` or be built-in (`sqlc/db-prepare`).

## plugins

```yaml
plugins:
  # WASM plugin (sandboxed, recommended)
  - name: "sqlc-gen-typescript"
    wasm:
      url: "https://github.com/.../plugin.wasm"
      sha256: "abc123..."
    env:
      - OPTIONAL_ENV_VAR

  # Process plugin (runs local command)
  - name: "sqlc-gen-json"
    process:
      cmd: "sqlc-gen-json"
      format: "json" # "json" or "protobuf" (default)
    env:
      - PATH
```

`SQLC_VERSION` is always available in the plugin environment.

## rules

```yaml
rules:
  - name: "no-select-star"
    message: "Do not use SELECT *"
    rule: "query.sql.contains('SELECT *')"

  - name: "require-where-on-delete"
    message: "DELETE must have WHERE clause"
    rule: |
      query.cmd == "exec" &&
      query.sql.contains("DELETE") &&
      !query.sql.contains("WHERE")
```

Rules use CEL (Common Expression Language). See [vet.md](vet.md) for the full CEL API.

## overrides (global)

Global overrides apply to all SQL sets:

```yaml
overrides:
  go:
    rename:
      id: "Identifier"
    overrides:
      - db_type: "pg_catalog.timestamptz"
        nullable: true
        engine: "postgresql"
        go_type:
          import: "time"
          type: "Time"
          pointer: true
```

## Complete example

```yaml
version: "2"
sql:
  - name: "users-service"
    schema: "sql/schema/"
    queries: "sql/queries/"
    engine: "postgresql"
    database:
      uri: "${DATABASE_URL}"
    gen:
      go:
        package: "db"
        out: "internal/db"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_interface: true
        emit_empty_slices: true
        query_parameter_limit: 3
        json_tags_case_style: "camel"
        overrides:
          - db_type: "uuid"
            go_type: "github.com/google/uuid.UUID"
          - db_type: "timestamptz"
            go_type:
              import: "time"
              type: "Time"
    rules:
      - sqlc/db-prepare
      - no-select-star

rules:
  - name: no-select-star
    message: "Avoid SELECT * — list columns explicitly"
    rule: "query.sql.contains('SELECT *')"
```
