# pgxgen

[![Go Reference](https://pkg.go.dev/badge/github.com/tkcrm/pgxgen.svg)](https://pkg.go.dev/github.com/tkcrm/pgxgen)
[![Go Version](https://img.shields.io/badge/go-1.26-blue)](https://go.dev/)
[![Go Report Card](https://goreportcard.com/badge/github.com/tkcrm/pgxgen)](https://goreportcard.com/report/github.com/tkcrm/pgxgen)
[![License](https://img.shields.io/github/license/tkcrm/pgxgen)](LICENSE)

Code generation tool for PostgreSQL, MySQL, and SQLite. Generates CRUD SQL, Go models, and [sqlc](https://github.com/sqlc-dev/sqlc) query code from a single config file.

## Features

- **One config** — `pgxgen.yaml` replaces both `pgxgen.yaml` and `sqlc.yaml`
- **CRUD generation** — template-based SQL with soft delete, batch insert support
- **Go models** — structs with json/db tags, custom tags, type overrides, enum generation
- **sqlc integration** — auto-generates `sqlc.yaml` and runs sqlc
- **Multi-engine** — PostgreSQL + MySQL + SQLite in one project
- **Per-table repos** or **single repo** layout
- **Schema dump** — consolidated DDL output from migrations (no config needed)
- **SQL formatting** — format SQL files with dialect support (PostgreSQL, MySQL, SQLite)
- **Watch mode**, **dry-run**, **validation**, **interactive init**

## AI Agent Skills

This repository includes [AI agent skills](https://github.com/sxwebdev/skills) with documentation and usage examples for all packages. Install them with the [skills](https://github.com/sxwebdev/skills) CLI:

```bash
go install github.com/sxwebdev/skills/cmd/skills@latest
skills init
skills repo add tkcrm/pgxgen -all
```

## Install

```bash
go install github.com/tkcrm/pgxgen/cmd/pgxgen@latest
```

Or from source:

```bash
git clone https://github.com/tkcrm/pgxgen.git
cd pgxgen && go build -o pgxgen ./cmd/pgxgen
```

## Quick start

```bash
# Create config interactively
pgxgen init

# Or generate an example config
pgxgen example > pgxgen.yaml

# Generate everything
pgxgen generate

# Preview without writing
pgxgen generate --dry-run
```

## Configuration

```yaml
version: "2"

schemas:
  - name: main
    engine: postgresql # postgresql | mysql | sqlite
    schema_dir: sql/migrations

    # Go model generation
    models:
      output_dir: internal/models
      output_file_name: models_gen.go # default: models.go
      package_name: models
      package_path: github.com/your-org/project/internal/models
      sql_package: pgx/v5 # pgx/v5 | pgx/v4 | database/sql
      emit_json_tags: true
      emit_db_tags: true
      emit_pointers_for_null: false
      include_struct_comments: false # adds // @name StructName for Swagger
      custom_types: [MyCustomType] # types defined in models package
      skip_tables: [migrations, schema_version]
      skip_enums: [internal_status]
      type_overrides:
        - sql_type: uuid
          go_type: github.com/google/uuid.UUID
          import: github.com/google/uuid
      custom_tags:
        - name: validate
          format: required # applied to NOT NULL columns

    # sqlc auto-generation (pgxgen generates sqlc.yaml automatically)
    sqlc:
      keep_generated_config: false # keep .pgxgen/sqlc.yaml after run
      defaults:
        sql_package: pgx/v5 # pgx/v5 | pgx/v4 | database/sql
        emit_prepared_queries: false
        emit_interface: true
        emit_json_tags: true
        emit_db_tags: true
        emit_exported_queries: false
        emit_exact_table_names: false
        emit_empty_slices: true
        emit_result_struct_pointers: true
        emit_params_struct_pointers: false
        emit_enum_valid_method: true
        emit_all_enum_values: true
        query_parameter_limit: 1
        json_tags_case_style: camel # camel | pascal | snake | none
      overrides:
        rename: { d: Params }
        types:
          - db_type: uuid
            go_type: github.com/google/uuid.UUID
          - db_type: uuid
            nullable: true
            go_type: github.com/google/uuid.NullUUID
          # Object form (with explicit import):
          # - db_type: jsonb
          #   go_type:
          #     type: MyJSON
          #     import: github.com/your-org/project/internal/types
        columns:
          - column: users.email
            go_struct_tag: 'validate:"required,email"'
          # - column: users.metadata
          #   go_type: github.com/your-org/project/internal/types.Metadata

    # Defaults inherited by all tables in this schema
    defaults:
      # Pattern A — per-table repos (each table gets its own directory):
      queries_dir_prefix: sql/queries # → sql/queries/{table}
      output_dir_prefix: internal/store/repos # → internal/store/repos/{table}
      package_prefix: "" # optional: prepended to table name for package
      package_suffix: "" # optional: appended to table name for package

      # Pattern B — single repo (use instead of *_prefix above):
      # queries_dir: sql/queries
      # output_dir: internal/store

      crud:
        auto_clean: true
        exclude_table_name: true # GetByID instead of GetUserByID
        methods:
          create:
            skip_columns: [id, updated_at]
            returning: "*"
            column_values: { created_at: "now()" }
      constants:
        include_column_names: true

    # Per-table settings (crud + constants + sqlc + soft_delete)
    tables:
      users:
        primary_column: id
        # queries_dir: sql/queries/custom    # override defaults for this table
        # output_dir: internal/store/custom  # override defaults for this table
        # sqlc:
        #   query_parameter_limit: 3
        # constants:
        #   include_column_names: false
        # soft_delete:
        #   column: deleted_at
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
              order: { by: created_at, direction: DESC } # direction default: DESC
              limit: true
              where_additional: ["status = 'active'"] # extra raw SQL conditions
            total: {}
            exists:
              where:
                email: {} # default: = $1
                age: { operator: ">=", value: "$2" } # custom operator/value
            # batch_create:
            #   skip_columns: [id, created_at]

    # Custom SQL queries (free-form, generated alongside CRUD)
    custom_queries:
      - name: GetActiveUsers
        type: many # one | many | exec | copyfrom
        table: users # optional: ties query to a table's output_dir
        # output_dir: internal/store/reports  # optional override
        sql: |
          SELECT * FROM users
          WHERE is_active = true
          ORDER BY created_at DESC

# Custom templates (optional, override built-in templates)
templates:
  crud_dir: .pgxgen/templates/crud
  models_dir: .pgxgen/templates/models
```

### Path patterns

**Per-table repos** (each table gets its own directory):

```yaml
defaults:
  queries_dir_prefix: sql/queries # → sql/queries/{table}
  output_dir_prefix: internal/store/repos # → internal/store/repos/{table}
```

**Single repo** (all tables in one directory):

```yaml
defaults:
  queries_dir: sql/queries
  output_dir: internal/store
```

### CRUD methods

| Method         | SQL                                | sqlc type                          |
| -------------- | ---------------------------------- | ---------------------------------- |
| `create`       | INSERT                             | `:one` (with returning) or `:exec` |
| `update`       | UPDATE                             | `:one` (with returning) or `:exec` |
| `delete`       | DELETE (or UPDATE for soft delete) | `:exec`                            |
| `get`          | SELECT ... LIMIT 1                 | `:one`                             |
| `find`         | SELECT with WHERE, ORDER, LIMIT    | `:many`                            |
| `total`        | SELECT count(1)                    | `:one`                             |
| `exists`       | SELECT EXISTS                      | `:one`                             |
| `batch_create` | INSERT (copyfrom/multi-values)     | `:copyfrom`                        |

### Soft delete

```yaml
tables:
  posts:
    soft_delete:
      column: deleted_at
    crud:
      methods:
        delete: {} # → UPDATE SET deleted_at = now() WHERE id = $1
        get: {} # → auto adds WHERE deleted_at IS NULL
        find: {} # → auto adds WHERE deleted_at IS NULL
```

### Schema-qualified table names

For tables in non-default schemas, use dot notation in the table key. The migration files must include `CREATE SCHEMA` before creating objects in that schema:

```sql
-- sql/migrations/001_init.sql
CREATE SCHEMA shop;

CREATE TABLE users ( id UUID PRIMARY KEY, email TEXT NOT NULL );
CREATE TABLE shop.orders ( id UUID PRIMARY KEY, total NUMERIC NOT NULL );

CREATE TYPE shop.order_status AS ENUM ('pending', 'shipped', 'delivered');
```

```yaml
tables:
  users:                    # public schema (default)
    primary_column: id
    crud:
      methods:
        get: {}
        create: { returning: "*" }

  shop.orders:              # "shop" schema
    primary_column: id
    crud:
      methods:
        get: {}
        create: { returning: "*" }
```

When a schema prefix is specified:

- **SQL** uses the schema-qualified name: `INSERT INTO shop.orders`, `SELECT * FROM shop.orders`
- **Go identifiers** are prefixed with the schema name: `CreateShopOrder`, `GetShopOrder`
- **File paths** are prefixed with the schema name: `sql/queries/shop_orders/shop_orders_gen.sql`
- **Go types** (models, enums) are prefixed to match sqlc's naming convention: `shop.orders` → `ShopOrder`, `shop.order_status` → `ShopOrderStatus`
- **Cross-schema references** are resolved correctly — a column of type `shop.order_status` in any schema resolves to `ShopOrderStatus`

This ensures no collisions when the same table name exists in different schemas. Tables in the default schema (`public` for PostgreSQL, `main` for SQLite) remain unprefixed.

## Commands

| Command                     | Description                                            |
| --------------------------- | ------------------------------------------------------ |
| `pgxgen generate`           | Generate everything (crud + models + sqlc + constants) |
| `pgxgen generate crud`      | Generate CRUD SQL only                                 |
| `pgxgen generate models`    | Generate Go models only                                |
| `pgxgen generate --dry-run` | Preview changes without writing                        |
| `pgxgen schema <path>`      | Output consolidated DDL from migrations                |
| `pgxgen fmt <path>`         | Format SQL files (with confirmation)                   |
| `pgxgen fmt <path> --check` | Check SQL formatting (for CI, exit 1 if unformatted)   |
| `pgxgen validate`           | Validate config and schema (for CI)                    |
| `pgxgen watch`              | Watch schema files, regenerate on changes              |
| `pgxgen init`               | Create config interactively                            |
| `pgxgen example`            | Print example config with all features                 |
| `pgxgen migrate`            | Migrate v1 config to v2                                |
| `pgxgen update`             | Self-update to latest version                          |

## Schema dump

Output the final consolidated DDL from all migration files — as if all migrations were applied and the schema was dumped. This command is standalone and does not require a `pgxgen.yaml` config.

```bash
pgxgen schema sql/migrations/postgres                  # PostgreSQL (default)
pgxgen schema sql/migrations/sqlite -e sqlite          # SQLite
pgxgen schema sql/migrations/postgres > schema.sql     # Save to file
pgxgen schema sql/migrations/001_init.up.sql           # Single file
```

Captures tables, columns with defaults, PRIMARY KEY, FOREIGN KEY, UNIQUE, CHECK constraints, indexes (including partial), extensions, enums, and comments. Tables are ordered by foreign key dependencies.

## SQL formatting

Format SQL files with dialect-aware formatting. Shows files to format and asks for confirmation before writing. This command is standalone and does not require a `pgxgen.yaml` config.

```bash
pgxgen fmt .                                    # Format all .sql files recursively
pgxgen fmt sql/migrations                       # Format directory
pgxgen fmt sql/migrations/001_init.up.sql       # Format single file
pgxgen fmt . --check                            # Check only (for CI)
pgxgen fmt . --dry-run                          # Process without saving (test)
pgxgen fmt . --yes                              # Skip confirmation
pgxgen fmt . -e mysql                           # MySQL dialect
```

Supports PostgreSQL (default), MySQL, and SQLite dialects. Preserves comments, handles dollar-quoting (PostgreSQL), backtick identifiers (MySQL).

## Migration from v1

```bash
pgxgen migrate --in-place --sqlc-config sqlc.yaml
rm sqlc.yaml
pgxgen generate
```

## JSON Schema

IDE autocompletion is available via [schemas/pgxgen-schema.json](schemas/pgxgen-schema.json):

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/tkcrm/pgxgen/master/schemas/pgxgen-schema.json
version: "2"
```

## License

[MIT](LICENSE)
