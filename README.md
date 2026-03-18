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
- **Watch mode**, **dry-run**, **validation**, **interactive init**

## AI Agent Skills

This repository includes [AI agent skills](https://github.com/sxwebdev/skills) with documentation and usage examples for all packages. Install them with the [skills](https://github.com/sxwebdev/skills) CLI:

```bash
go install github.com/sxwebdev/skills/cmd/skills@latest
skills init
skills repo add tkcrm/pgxgen
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
    engine: postgresql
    schema_dir: sql/migrations

    models:
      output_dir: internal/models
      output_file_name: models_gen.go
      package_name: models
      package_path: github.com/your-org/project/internal/models
      emit_json_tags: true
      emit_db_tags: true

    sqlc:
      defaults:
        sql_package: pgx/v5
        emit_interface: true
        emit_json_tags: true
        emit_db_tags: true
        emit_empty_slices: true
        emit_result_struct_pointers: true
        emit_enum_valid_method: true
        emit_all_enum_values: true
      overrides:
        rename: { d: Params }
        types:
          - db_type: uuid
            go_type: "github.com/google/uuid.UUID"
          - db_type: uuid
            nullable: true
            go_type: "github.com/google/uuid.NullUUID"
        columns:
          - column: users.email
            go_struct_tag: 'validate:"required,email"'

    defaults:
      queries_dir_prefix: sql/queries
      output_dir_prefix: internal/store/repos
      crud:
        auto_clean: true
        exclude_table_name: true
        methods:
          create:
            skip_columns: [id, updated_at]
            returning: "*"
            column_values: { created_at: "now()" }
      constants:
        include_column_names: true

    tables:
      users:
        primary_column: id
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
              order: { by: created_at, direction: DESC }
              limit: true
            total: {}
            exists:
              where: { email: {} }
```

### Path patterns

**Per-table repos** (each table gets its own directory):

```yaml
defaults:
  queries_dir_prefix: sql/queries # → sql/queries/{table}
  output_dir_prefix: internal/store/repos # → internal/store/repos/repo_{table}
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

## Commands

| Command                     | Description                                            |
| --------------------------- | ------------------------------------------------------ |
| `pgxgen generate`           | Generate everything (crud + models + sqlc + constants) |
| `pgxgen generate crud`      | Generate CRUD SQL only                                 |
| `pgxgen generate models`    | Generate Go models only                                |
| `pgxgen generate --dry-run` | Preview changes without writing                        |
| `pgxgen validate`           | Validate config and schema (for CI)                    |
| `pgxgen watch`              | Watch schema files, regenerate on changes              |
| `pgxgen init`               | Create config interactively                            |
| `pgxgen example`            | Print example config with all features                 |
| `pgxgen migrate`            | Migrate v1 config to v2                                |
| `pgxgen update`             | Self-update to latest version                          |

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
