---
name: sqlc
description: >
  Compile SQL queries to type-safe Go code with sqlc. Use this skill whenever working with sqlc configuration
  (sqlc.yaml, sqlc.json), writing SQL query files with sqlc annotations (:one, :many, :exec, :execrows,
  :execresult, :execlastid, :copyfrom, :batchone, :batchmany, :batchexec), configuring type overrides,
  setting up vet/lint rules, managing sqlc plugins (WASM or process), or troubleshooting sqlc code generation.
  Also triggers when code imports generated sqlc packages, when the user mentions "sqlc generate",
  "sqlc init", "sqlc vet", "sqlc diff", "sqlc compile", or asks about SQL-to-Go code generation,
  query compilation, database schema parsing, or Go struct generation from SQL tables.
  Applies to PostgreSQL, MySQL, and SQLite engines.
user-invocable: false
---

# sqlc — Compile SQL to type-safe Go code

## Overview

sqlc generates fully type-safe Go code from SQL queries. The workflow is:

1. Write SQL schema (CREATE TABLE, types, enums)
2. Write SQL queries with name annotations
3. Run `sqlc generate`
4. Get type-safe Go code: models, query methods, interfaces

sqlc parses SQL at compile time — no reflection, no runtime query building. This catches
SQL errors before the code runs and produces Go code that reads like hand-written database access.

## Core workflow

### 1. Initialize a project

```bash
sqlc init
```

This creates an empty `sqlc.yaml` with version 2 format. See [references/config.md](references/config.md)
for the full configuration reference.

### 2. Write schema SQL

```sql
-- schema.sql
CREATE TABLE authors (
    id   BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    bio  TEXT
);
```

Schema can be a single file, a directory of files, or a list of paths. sqlc also supports
migration files from golang-migrate, goose, atlas, dbmate, sql-migrate, and tern — it
automatically ignores down migrations.

### 3. Write annotated queries

```sql
-- query.sql

-- name: GetAuthor :one
SELECT * FROM authors
WHERE id = $1 LIMIT 1;

-- name: ListAuthors :many
SELECT * FROM authors
ORDER BY name;

-- name: CreateAuthor :one
INSERT INTO authors (name, bio)
VALUES ($1, $2)
RETURNING *;

-- name: DeleteAuthor :exec
DELETE FROM authors WHERE id = $1;
```

Each query needs a `-- name: <Name> :<command>` comment. See [references/queries.md](references/queries.md)
for all annotation types and macros.

### 4. Configure sqlc.yaml

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
        sql_package: "pgx/v5"
```

### 5. Generate code

```bash
sqlc generate
```

This produces:

- `db.go` — DBTX interface, `New()` constructor, `Queries` struct
- `models.go` — Go structs for tables and enums
- `query.sql.go` — Type-safe methods for each query

## Instructions

### When helping with configuration

Read [references/config.md](references/config.md) for the complete v2 config reference.

Key decisions to help the user with:

- **Engine**: `postgresql`, `mysql`, or `sqlite`
- **SQL package**: `pgx/v5` (recommended for PostgreSQL), `database/sql`, `pgx/v4`
- **Output options**: `emit_json_tags`, `emit_db_tags`, `emit_interface`, etc.
- **Type overrides**: Map SQL types to custom Go types. See [references/overrides.md](references/overrides.md)

Always use version `"2"` format. Version 1 is legacy and should not be used.

### When helping with queries

Read [references/queries.md](references/queries.md) for all annotation types and macros.

Key points:

- PostgreSQL uses `$1, $2` for parameters; MySQL/SQLite use `?`
- Use `sqlc.arg('name')` or `@name` (PostgreSQL only) to name parameters
- Use `sqlc.narg('name')` for nullable parameters
- Use `sqlc.embed(table)` to embed a table struct in results
- Use `sqlc.slice('name')` for dynamic IN clauses (MySQL/SQLite)
- Batch operations (`:batchone`, `:batchmany`, `:batchexec`) require pgx

### When helping with type overrides

Read [references/overrides.md](references/overrides.md) for override patterns.

Common scenarios:

- UUID columns → `github.com/google/uuid.UUID`
- JSON/JSONB columns → custom struct types
- Nullable types → pointer types or `sql.Null*` types
- Timestamp types → `time.Time` with specific handling

### When helping with vet rules

Read [references/vet.md](references/vet.md) for rule configuration.

- Use `sqlc/db-prepare` built-in rule to validate against a real database
- Write custom CEL rules to enforce query patterns
- Disable rules per-query with `/* @sqlc-vet-disable rule-name */`

### When helping with plugins

Read [references/plugins.md](references/plugins.md) for plugin configuration.

- WASM plugins are sandboxed and recommended
- Process plugins run arbitrary commands — use for local development tools
- Plugins extend code generation beyond Go

### When helping with CLI commands

Read [references/cli.md](references/cli.md) for all commands.

Common commands:

- `sqlc generate` — Generate code
- `sqlc compile` — Check SQL without generating
- `sqlc vet` — Run lint rules
- `sqlc diff` — Compare generated vs existing code (CI use)

### When troubleshooting

Common issues:

1. **"relation does not exist"** — Schema file not included or wrong path in config
2. **"column does not exist"** — Typo in query or schema not matching
3. **Type mismatch errors** — Add an override for the problematic type
4. **"no queries found"** — Missing or malformed annotation comment
5. **Parameter count mismatch** — Check `$1, $2` numbering (PostgreSQL) or `?` count (MySQL)

## Examples

**Example 1: User asks to set up sqlc for a new PostgreSQL project**

Create the config and example files:

```yaml
# sqlc.yaml
version: "2"
sql:
  - schema: "sql/schema/"
    queries: "sql/queries/"
    engine: "postgresql"
    gen:
      go:
        package: "db"
        out: "internal/db"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_interface: true
        query_parameter_limit: 3
```

**Example 2: User wants to add a UUID type override**

```yaml
version: "2"
sql:
  - schema: "schema.sql"
    queries: "queries/"
    engine: "postgresql"
    gen:
      go:
        package: "db"
        out: "internal/db"
        sql_package: "pgx/v5"
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

**Example 3: User wants batch insert for PostgreSQL**

```sql
-- name: CreateAuthors :batchexec
INSERT INTO authors (name, bio) VALUES ($1, $2);
```

This generates a `BatchResults` type with `Exec(func(int, error))` — requires pgx.

**Example 4: User wants to add a vet rule preventing SELECT \***

```yaml
version: "2"
sql:
  - schema: "schema.sql"
    queries: "queries/"
    engine: "postgresql"
    gen:
      go:
        package: "db"
        out: "internal/db"
        sql_package: "pgx/v5"
    rules:
      - no-select-star
rules:
  - name: no-select-star
    message: "Avoid SELECT * — list columns explicitly"
    rule: "query.sql.contains('SELECT *')"
```

## Key principles

- **Always use v2 config** — version 1 is legacy and lacks features like plugins, vet rules, and cloud support.
- **Match engine to database** — PostgreSQL features (arrays, enums, RETURNING) don't work with MySQL/SQLite engines.
- **Prefer pgx/v5 for PostgreSQL** — It provides native type support (UUID, arrays, JSON) without extra overrides.
- **Keep queries simple** — sqlc parses SQL statically. Dynamic queries (conditional WHERE clauses) need separate query definitions.
- **Use overrides sparingly** — Only when the default type mapping doesn't fit. Over-customizing makes the config hard to maintain.
- **Run sqlc compile in CI** — Catches SQL errors without needing a database. Use `sqlc diff` to ensure generated code is committed.

## Generated code structure

| File           | Contains                                        | Always generated                  |
| -------------- | ----------------------------------------------- | --------------------------------- |
| `db.go`        | DBTX interface, New(), Queries struct, WithTx() | Yes                               |
| `models.go`    | Table structs, enum types and constants         | Yes                               |
| `query.sql.go` | SQL constants, Params structs, query methods    | Yes (per query file)              |
| `querier.go`   | Querier interface (all query methods)           | Only with `emit_interface: true`  |
| `batch.go`     | Batch operation types                           | Only with `:batch*` annotations   |
| `copyfrom.go`  | COPY FROM helpers                               | Only with `:copyfrom` annotations |
