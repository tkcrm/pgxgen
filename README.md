# pgxgen

pgxgen use [`sqlc`](https://github.com/kyleconroy/sqlc) tool with additional improvements.

- Instead of `database/sql` used [`pgx/v4`](https://github.com/jackc/pgx)
- Instead null types like `sql.NullString` used nil type `*string`
- Auto generate CRUD for existing tables in postgresql database

## Install

```bash
go install github.com/tkcrm/pgxgen/cmd/pgxgen@latest
```

## Usage

### Generate `CRUD` queries for existing tables

```bash
pgxgen gencrud -c=postgres://DB_USER:DB_PASSWD@DB_HOST:DB_PORT/DB_NAME?sslmode=disable
```

### Configure `sqlc`

At root of your project create a `sqlc.yaml` file with the configuration described below.

> Configuration available [there](https://docs.sqlc.dev/en/stable/reference/config.html)

#### Configuration file `sqlc.yaml` example

```yaml
version: 1
packages:
  - path: "./internal/store"
    name: "store"
    engine: "postgresql"
    schema: "migrations"
    queries: "sql"
    sql_package: "pgx/v4" # REQUIRED!
    emit_json_tags: true
    emit_exported_queries: true
    emit_db_tags: true
    emit_interface: true
    emit_exact_table_names: false
    emit_empty_slices: true
    emit_result_struct_pointers: true
```

> **NOTICE!** Option `sql_package: "pgx/v4"` is required in configuration file

### Generate `db`, `models` and `interface`

```bash
pgxgen generate
```
