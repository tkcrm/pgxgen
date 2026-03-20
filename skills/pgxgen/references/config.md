# pgxgen v2 Configuration Reference

## Table of Contents

- [Root Config](#root-config)
- [Schema Config](#schema-config)
- [Models Config](#models-config)
- [sqlc Config](#sqlc-config)
- [Defaults Config](#defaults-config)
- [Table Config](#table-config)
- [CRUD Methods](#crud-methods)
- [Soft Delete](#soft-delete)
- [Custom Queries](#custom-queries)
- [Templates](#templates)
- [Full Example](#full-example)

## Root Config

```yaml
version: "2" # Required. Must be "2"
schemas: [...] # Required. Min 1 schema
templates: # Optional custom template dirs
  crud_dir: path
  models_dir: path
```

## Schema Config

```yaml
schemas:
  - name: main # Required. Human-readable name
    engine: postgresql # Required. postgresql | mysql | sqlite
    schema_dir: sql/migrations # Required. Path to .sql migration files
    models: { ... } # Optional. Go model generation
    sqlc: { ... } # Optional. sqlc integration
    defaults: { ... } # Optional. Defaults for all tables
    tables: { ... } # Optional. Per-table config
    custom_queries: [...] # Optional. Hand-written SQL queries
```

## Models Config

```yaml
models:
  output_dir: internal/models # Required. Where to write models_gen.go
  output_file_name: models_gen.go # Optional. Default: "models.go"
  package_name: models # Required. Go package name
  package_path: github.com/org/proj/internal/models # Optional. Full import path
  emit_json_tags: true # Optional. Add json struct tags
  emit_db_tags: true # Optional. Add db struct tags
  emit_pointers_for_null: false # Optional. Use *type for nullable (pgx only)
  include_struct_comments: false # Optional. Add // @name comments for Swagger
  sql_package: pgx/v5 # Optional. pgx/v5 | pgx/v4 | database/sql
  custom_types: [MyType] # Optional. Types defined in models package
  type_overrides: # Optional. SQL → Go type overrides
    - sql_type: uuid
      go_type: uuid.UUID
      import: github.com/google/uuid # Optional import path
  custom_tags: # Optional. Additional struct tags
    - name: validate
      format: required # Applied to NOT NULL columns
```

## sqlc Config

pgxgen auto-generates `.pgxgen/sqlc.yaml` from this section.

```yaml
sqlc:
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
    query_parameter_limit: 3 # Optional int
    json_tags_case_style: snake # Optional

  overrides:
    rename: # Rename sqlc-generated identifiers
      d: Params
    types: # db_type → go_type mapping
      - db_type: uuid
        go_type: "github.com/google/uuid.UUID"
      - db_type: uuid
        nullable: true
        go_type: "github.com/google/uuid.NullUUID"
    columns: # Per-column overrides
      - column: users.email
        go_struct_tag: 'validate:"required,email"'
      - column: orders.amount
        go_type: "github.com/shopspring/decimal.Decimal"
```

## Defaults Config

```yaml
defaults:
  # Pattern A: per-table repos (each table gets its own directory)
  queries_dir_prefix: sql/queries # → sql/queries/{table}/
  output_dir_prefix: internal/store/repos # → internal/store/repos/{table}/

  # Pattern B: single repo (all tables in one directory)
  # queries_dir: sql/queries
  # output_dir: internal/store

  crud:
    auto_clean: true # Remove old _gen.sql before regenerating
    exclude_table_name: true # GetByID instead of GetUserByID
    methods: # Default methods for all tables
      create:
        skip_columns: [id, updated_at]
        returning: "*"
        column_values: { created_at: "now()" }

  constants:
    include_column_names: true # Generate column name constants
```

Use `queries_dir_prefix` + `output_dir_prefix` (Pattern A) OR `queries_dir` + `output_dir` (Pattern B). Do not mix.

## Table Config

```yaml
tables:
  users:
    primary_column: id # Primary key column. Default: "id"
    queries_dir: sql/queries/custom_path # Override queries dir for this table
    output_dir: internal/store/custom # Override output dir for this table

    soft_delete:
      column: deleted_at # Enable soft delete on this column

    crud:
      methods:
        create: { ... }
        update: { ... }
        delete: {}
        get: { ... }
        find: { ... }
        total: {}
        exists: { ... }
        batch_create: { ... }

    constants:
      include_column_names: true # Override default for this table

    sqlc:
      query_parameter_limit: 5 # Override sqlc param limit for this table
```

Only methods listed in `crud.methods` are generated. An empty `{}` uses defaults.

## CRUD Methods

See `references/crud-methods.md` for detailed method configuration.

## Soft Delete

When `soft_delete.column` is set:

- `delete` generates `UPDATE SET {column} = now()` instead of `DELETE`
- `get` and `find` auto-add `WHERE {column} IS NULL`

```yaml
tables:
  posts:
    soft_delete:
      column: deleted_at
    crud:
      methods:
        delete: {} # → UPDATE posts SET deleted_at = now() WHERE id = $1
        get: {} # → SELECT * FROM posts WHERE id = $1 AND deleted_at IS NULL
        find: {} # → SELECT * FROM posts WHERE deleted_at IS NULL ...
```

## Custom Queries

Hand-written SQL queries that pgxgen passes to sqlc:

```yaml
custom_queries:
  - name: GetActiveUsers
    type: many # one | many | exec | copyfrom
    table: users # Optional. Associated table
    output_dir: internal/store/repos/users # Optional. Override output
    sql: |
      SELECT * FROM users
      WHERE is_active = true
      ORDER BY created_at DESC
```

## Templates

Override built-in CRUD or model templates:

```yaml
templates:
  crud_dir: .pgxgen/templates/crud # Custom CRUD SQL templates
  models_dir: .pgxgen/templates/models # Custom model templates
```

Custom CRUD templates should follow the engine subdirectory structure:
`{crud_dir}/{engine}/{method}.sql.tmpl` (e.g., `postgresql/create.sql.tmpl`).

## Full Example

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/tkcrm/pgxgen/master/schemas/pgxgen-schema.json
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

      posts:
        primary_column: id
        soft_delete:
          column: deleted_at
        crud:
          methods:
            create:
              skip_columns: [id, updated_at, deleted_at]
              column_values: { created_at: "now()" }
              returning: "*"
            update:
              skip_columns: [id, created_at, deleted_at]
              column_values: { updated_at: "now()" }
              returning: "*"
            get: {}
            delete: {}
            find:
              order: { by: created_at, direction: DESC }
              limit: true
```
