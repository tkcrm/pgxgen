---
name: pgxgen
description: >
  Configure and use pgxgen — a code generation tool for PostgreSQL, MySQL, and SQLite.
  Triggers when: working with pgxgen.yaml config files, generating CRUD SQL queries,
  generating Go models from SQL schema, sqlc integration and auto-generated sqlc.yaml,
  database code generation, Go struct generation from tables, type mapping (SQL to Go),
  soft delete configuration, batch inserts, table constants generation, or any mention
  of pgxgen CLI commands (generate, validate, watch, init, migrate, example, schema, fmt).
  Also triggers for: configuring per-table repos vs single repo layout, custom CRUD
  templates, nullable type handling (pgtype, sql.Null*), enum generation, and
  struct tag customization (json, db, validate tags).
user-invocable: true
---

# pgxgen

## Overview

pgxgen generates CRUD SQL, Go models, and sqlc query code from a single `pgxgen.yaml` config file. It replaces the need for separate `pgxgen.yaml` and `sqlc.yaml` files. It supports PostgreSQL, MySQL, and SQLite in one project.

**Generation pipeline:**

1. Parse SQL schema files (`.sql`) into an in-memory catalog
2. Generate CRUD SQL queries from templates (per engine)
3. Generate Go model structs with configurable tags and type mapping
4. Auto-generate `sqlc.yaml` and run sqlc for query code
5. Generate Go constants for table/column names

**Key source files:**

- Config types: `internal/config/v2.go`
- Orchestrator: `internal/codegen/orchestrator.go`
- CRUD generator: `internal/codegen/crud/generator.go`
- Models generator: `internal/codegen/models/generator.go`
- SQL parsers: `internal/sqlparser/{postgresql,mysql,sqlite}.go`
- Type mappers: `internal/sqlparser/typemap/{postgresql,mysql,sqlite}.go`
- CLI: `internal/cli/app.go`
- CRUD templates: `internal/codegen/crud/templates/{postgresql,mysql,sqlite}/`

## Instructions

### When the user asks about pgxgen configuration

1. Read `references/config.md` for the complete v2 config reference
2. Check the user's existing `pgxgen.yaml` if present
3. Suggest config based on their database engine and project layout

### When the user asks about CRUD methods

1. Read `references/crud-methods.md` for all method types and options
2. Each CRUD method (create, update, delete, get, find, total, exists, batch_create) has specific options
3. Soft delete changes behavior of delete/get/find methods automatically

### When the user asks about type mapping or Go models

1. Read `references/type-mapping.md` for SQL-to-Go type tables per engine
2. Type resolution priority: sqlc column overrides > sqlc type overrides > models.type_overrides > default typemap > pointer replacement
3. The `sql_package` setting (pgx/v5, pgx/v4, database/sql) affects nullable type mapping

### When the user asks about CLI commands

1. Read `references/cli.md` for all commands, flags, and usage
2. Default command is `pgxgen generate` (runs all generators)
3. `--dry-run` previews without writing, `--debug` shows timing
4. `pgxgen schema <path>` outputs consolidated DDL from migrations (standalone, no config needed)
5. `pgxgen fmt <path>` formats SQL files with confirmation prompt (standalone, no config needed)

### When the user needs to write or modify pgxgen.yaml

Always include the JSON schema reference at the top for IDE autocompletion:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/tkcrm/pgxgen/master/schemas/pgxgen-schema.json
version: "2"
```

### When the user asks about project layout

Two layout patterns exist:

**Per-table repos** (recommended for larger projects):

```yaml
defaults:
  queries_dir_prefix: sql/queries # → sql/queries/{table}/
  output_dir_prefix: internal/store/repos # → internal/store/repos/{table}/
  # package_prefix: repo_ # → internal/store/repos/repo_{table}/
  # package_suffix: ""
```

**Single repo** (simpler projects):

```yaml
defaults:
  queries_dir: sql/queries
  output_dir: internal/store
```

### When the user migrates from v1

```bash
pgxgen migrate --in-place --sqlc-config sqlc.yaml
rm sqlc.yaml
pgxgen generate
```

## Examples

**Example 1: User asks "Add a users table with CRUD to my pgxgen config"**

Read the existing `pgxgen.yaml`, then add a table entry:

```yaml
tables:
  users:
    primary_column: id
    crud:
      methods:
        create:
          skip_columns: [id, updated_at]
          column_values: { created_at: "now()" }
          returning: "*"
        update:
          skip_columns: [id, created_at]
          column_values: { updated_at: "now()" }
          returning: "*"
        get: { name: GetByID }
        delete: {}
        find:
          order: { by: created_at, direction: DESC }
          limit: true
        total: {}
```

**Example 2: User asks "How do I add soft delete to my posts table?"**

```yaml
tables:
  posts:
    primary_column: id
    soft_delete:
      column: deleted_at
    crud:
      methods:
        delete: {} # generates: UPDATE posts SET deleted_at = now() WHERE id = $1
        get: {} # auto-adds: WHERE deleted_at IS NULL
        find: {} # auto-adds: WHERE deleted_at IS NULL
```

**Example 3: User asks "Map uuid to google/uuid in my models"**

Two approaches — models-level or sqlc-level:

```yaml
# Option A: models type_overrides (affects Go model generation)
models:
  type_overrides:
    - sql_type: uuid
      go_type: uuid.UUID
      import: github.com/google/uuid

# Option B: sqlc overrides (affects sqlc-generated query code)
sqlc:
  overrides:
    types:
      - db_type: uuid
        go_type: "github.com/google/uuid.UUID"
      - db_type: uuid
        nullable: true
        go_type: "github.com/google/uuid.NullUUID"
```

## Key principles

- pgxgen v2 uses a single `pgxgen.yaml` — never suggest creating a separate `sqlc.yaml` because pgxgen auto-generates it in `.pgxgen/sqlc.yaml`
- Generated files use `_gen` suffix (`models_gen.go`, `{table}_gen.sql`, `constants_gen.go`) and include a "DO NOT EDIT" marker — never suggest editing generated files
- Default CRUD settings in `defaults.crud.methods` cascade to all tables; per-table `crud.methods` override specific methods
- The `returning` option only works with PostgreSQL (other engines use `:exec` instead of `:one`)
- `auto_clean: true` removes old generated SQL files before regenerating — safe for CI
- When `exclude_table_name: true`, method names become `GetByID` instead of `GetUserByID`
