# pgxgen

pgxgen use [`sqlc`](https://github.com/kyleconroy/sqlc) tool with additional improvements.

- Prefere using [`pgx/v4`](https://github.com/jackc/pgx) instead of `database/sql`
- Instead null types like `sql.NullString` used nil type `*string`
- Auto generate CRUD for existing tables in postgresql database

## Install

```bash
go install github.com/tkcrm/pgxgen/cmd/pgxgen@latest
```

## Usage

### Configure `pgxgen`

At root of your project create a `pgxgen.yaml`. Example of configuration below.

```yaml
version: 1
pgxgen:
  # Result SQL file name; default: crud_queries.sql
  # Will save to `queries` path from `sqlc.yaml` config
  output_crud_sql_file_name: "crud_queries.sql"
  crud_params:
    # Limit and offset for `Find` method
    limit: true
    # Order by for `Find` method
    order_by:
      - by: id
        order: desc
        tables:
          # List tables or all (*)
          - "*"
    where:
      # g - get
      # f - find
      # u - update
      # d - delete
      # t - total
      - methods: "fgtud"
        # List tables or all (*)
        tables:
          - users
        params:
          - organization_id
```

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
    sql_package: "pgx/v4"
    emit_prepared_queries: false
    emit_json_tags: true
    emit_exported_queries: false
    emit_db_tags: true
    emit_interface: true
    emit_exact_table_names: false
    emit_empty_slices: true
    emit_result_struct_pointers: true
    emit_params_struct_pointers: false
```

### Generate `db`, `models` and `interface`

```bash
pgxgen generate
```
