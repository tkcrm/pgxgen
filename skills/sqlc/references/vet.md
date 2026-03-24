# sqlc Vet Rules Reference

## Overview

`sqlc vet` runs lint rules against SQL queries. Rules are written in CEL (Common Expression Language)
and can check query structure, enforce naming conventions, or validate against a live database.

## Configuration

Define rules at the top level and reference them in SQL sets:

```yaml
version: "2"
sql:
  - schema: "schema.sql"
    queries: "queries/"
    engine: "postgresql"
    database:
      uri: "${DATABASE_URL}" # Required for sqlc/db-prepare
    gen:
      go:
        package: "db"
        out: "internal/db"
    rules:
      - sqlc/db-prepare # Built-in rule
      - no-select-star # Custom rule

rules:
  - name: no-select-star
    message: "Do not use SELECT * — list columns explicitly"
    rule: "query.sql.contains('SELECT *')"
```

## Built-in rules

### sqlc/db-prepare

Validates queries by preparing them against a real database. Catches:

- Invalid table/column references
- Type mismatches
- Syntax errors the parser might miss

Requires `database.uri` or `database.managed` in the SQL set config.

```yaml
sql:
  - database:
      uri: "postgresql://localhost:5432/mydb?sslmode=disable"
    rules:
      - sqlc/db-prepare
```

## CEL expression API

### Available variables

| Variable             | Type            | Description                                     |
| -------------------- | --------------- | ----------------------------------------------- |
| `config.engine`      | string          | `"postgresql"`, `"mysql"`, or `"sqlite"`        |
| `config.version`     | string          | Config version (`"2"`)                          |
| `config.schema`      | list(string)    | Schema file paths                               |
| `config.queries`     | list(string)    | Query file paths                                |
| `query.sql`          | string          | The full SQL text                               |
| `query.name`         | string          | Query name from annotation                      |
| `query.cmd`          | string          | Command type: `"one"`, `"many"`, `"exec"`, etc. |
| `query.params`       | list(Parameter) | Query parameters                                |
| `query.columns`      | list(Column)    | Result columns                                  |
| `postgresql.explain` | Explain         | PostgreSQL EXPLAIN output (requires database)   |
| `mysql.explain`      | Explain         | MySQL EXPLAIN output (requires database)        |

### Parameter object

| Field    | Type   | Description                    |
| -------- | ------ | ------------------------------ |
| `number` | int    | Parameter position (1-based)   |
| `name`   | string | Parameter name (from sqlc.arg) |

### Column object

| Field      | Type   | Description                |
| ---------- | ------ | -------------------------- |
| `name`     | string | Column name                |
| `not_null` | bool   | Whether column is NOT NULL |

### Explain object (PostgreSQL)

Available when `database.uri` is configured:

| Field                | Type   | Description          |
| -------------------- | ------ | -------------------- |
| `plan.node_type`     | string | Plan node type       |
| `plan.relation_name` | string | Table name           |
| `plan.total_cost`    | float  | Estimated total cost |
| `plan.plans`         | list   | Child plan nodes     |

## Custom rule examples

### Prevent SELECT \*

```yaml
- name: no-select-star
  message: "List columns explicitly instead of SELECT *"
  rule: "query.sql.contains('SELECT *')"
```

### Require WHERE on DELETE

```yaml
- name: delete-requires-where
  message: "DELETE must have a WHERE clause"
  rule: |
    !(query.cmd == "exec" &&
      query.sql.upper().contains("DELETE") &&
      !query.sql.upper().contains("WHERE"))
```

Note: Rule must evaluate to `true` to pass. Return `false` to fail.

### Limit query cost (requires database)

```yaml
- name: max-query-cost
  message: "Query cost exceeds threshold"
  rule: "postgresql.explain.plan.total_cost < 10000.0"
```

### Enforce naming convention

```yaml
- name: query-name-prefix
  message: "Query names must start with Get, List, Create, Update, or Delete"
  rule: |
    query.name.matches("^(Get|List|Create|Update|Delete|Upsert|Count|Search|Check|Find).*")
```

### Engine-specific rules

```yaml
- name: pg-only-rule
  message: "This rule only applies to PostgreSQL"
  rule: |
    config.engine != "postgresql" ||
    query.sql.contains("RETURNING")
```

## Disabling rules per query

Add a `@sqlc-vet-disable` comment to a query:

```sql
/* @sqlc-vet-disable */
-- name: LegacyQuery :many
SELECT * FROM old_table;
```

Disable specific rules:

```sql
/* @sqlc-vet-disable no-select-star delete-requires-where */
-- name: LegacyQuery :many
SELECT * FROM old_table;
```

The disable comment can span multiple lines:

```sql
/*
@sqlc-vet-disable
  no-select-star
  delete-requires-where
*/
-- name: LegacyQuery :many
SELECT * FROM old_table;
```

## Running vet

```bash
# Run all configured vet rules
sqlc vet

# With specific config file
sqlc vet -f sqlc.yaml
```

`sqlc vet` exits with non-zero status if any rule fails — suitable for CI.
