# pgxgen

[![Go Reference](https://pkg.go.dev/badge/github.com/tkcrm/pgxgen.svg)](https://pkg.go.dev/github.com/tkcrm/pgxgen)
[![Go Version](https://img.shields.io/badge/go-1.26-blue)](https://go.dev/)
[![Go Report Card](https://goreportcard.com/badge/github.com/tkcrm/pgxgen)](https://goreportcard.com/report/github.com/tkcrm/pgxgen)
[![License](https://img.shields.io/github/license/tkcrm/pgxgen)](LICENSE)

pgxgen uses [`sqlc`](https://github.com/sqlc-dev/sqlc) with additional improvements.

- Generate CRUD SQL for existing tables (PostgreSQL, MySQL, SQLite)
- Generate Go models for all tables from SQL schema
- Json tags: Omit empty and hide
- Update generated models with additional parameters: add / update fields and tags

> You can use [this repository](https://github.com/sxwebdev/pgxgen-example) which explains how to use `pgxgen` tool in your project

## Install

### Requirements

- `Go 1.26+`

### From Source Code

```bash
git clone https://github.com/tkcrm/pgxgen.git
cd pgxgen
go build -o bin/pgxgen ./cmd/pgxgen
sudo ./bin/pgxgen
```

### Or Install via go install

```bash
go install github.com/tkcrm/pgxgen/cmd/pgxgen@latest
```

### Or Install via script

```bash
/bin/bash -c "$(curl -fsSL 'https://raw.githubusercontent.com/tkcrm/pgxgen/refs/heads/master/scripts/install.sh')"
```

## Usage

```text
COMMANDS:
   crud               Generate crud sql's
   generate models    Generate Go models for all tables from SQL schema
   sqlc               Generate sqlc code
   update             Update pgxgen to the latest version
   version            Print the version
   help, h            Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --pgxgen-config value  Absolute or relative path to pgxgen.yaml file (default: "pgxgen.yaml")
   --sqlc-config value    Absolute or relative path to sqlc.yaml file (default: "sqlc.yaml")
   --help, -h             show help
   --version, -v          print the version
```

### Configure `pgxgen`

At root of your project create a `pgxgen.yaml`. Example of configuration below.

> You can specify a different name, but must use this flag: `--pgxgen-config [new_name.yaml]`
>
> Example: `pgxgen --pgxgen-config pgxgen-new.yaml`

```yaml
version: "1"
sqlc:
  - # directory with migrations. required
    schema_dir: sql/migrations
    models:
      # replace nullable types. ex: sql.NullInt32 -> *int32
      replace_sqlc_nullable_types: true
      # include comments for structs. useful for swagger generation
      include_struct_comments: false
      # move sqlc models to another package and directory
      move: # required
        output_dir: internal/models
        # default: models.go
        output_file_name: models_gen.go
        # new package name. by default based on `output_dir`
        package_name: models
        # required. full path to new models directory
        package_path: github.com/company/project/internal/models
        # optional. add custom imports to generated code by sqlc
        imports:
          - path: github.com/company/project/internal/models # required
            # optional. use path if this type detected in file
            go_type: MyStruct

    # generate crud sql for tables
    crud:
      # Auto remove generated files, ended with _gen.sql
      auto_remove_generated_files: true
      # Instead [ActionName][TableName] will be [ActionName]
      # Example GetUser -> Get; FindUsers -> Find, etc.
      # You can user `name` field for manual overwriting method name
      exclude_table_name_from_methods: false
      tables:
        user:
          # Not required. If you do not specify this value, then the sql file will be generated in each folder for all tables
          output_dir: sql/queries/users
          primary_column: id
          methods:
            # get
            # find
            # create
            # update
            # delete
            # total
            # exists
            create:
              skip_columns:
                - id
                - updated_at
              column_values:
                created_at: now()
              returning: "*"
            update:
              skip_columns:
                - id
                - created_at
              column_values:
                updated_at: now()
              returning: "*"
            find:
              where:
                user_id:
                  operator: "!="
                deleted_at:
                  value: "IS NULL"
              where_additional:
                - (NOT @is_is_active::boolean OR "is_active" = @is_active)
              order:
                by: created_at
                direction: DESC
              limit: true
            get:
              # Not required. By default this method will be GetUser
              name: GetUserByID
            delete:
            total:
            exists:
              where:
                email:

    # go constants
    constants:
      tables:
        users:
          output_dir: internal/store/users/repo_users
          include_column_names: true

# generate Go models for all tables from SQL schema
generate:
  models:
    - # directory with SQL schema/migration files. required
      schema_dir: sql/migrations
      # database engine: postgresql, mysql, sqlite. default: postgresql
      engine: postgresql
      # output directory for generated models. required
      output_dir: internal/models
      # output file name. default: models.go
      output_file_name: models.go
      # Go package name. required
      package_name: models
      # SQL driver package: pgx/v5, pgx/v4, database/sql. default: database/sql
      sql_package: pgx/v5
      # emit json struct tags
      emit_json_tags: true
      # emit db struct tags
      emit_db_tags: true
      # use pointers for nullable types instead of sql.Null*
      emit_pointers_for_null: false
```

### Configure `sqlc`

At root of your project create a `sqlc.yaml` file with the configuration described below.

> Configuration available [here](https://docs.sqlc.dev/en/stable/reference/config.html)

#### Configuration `sqlc.yaml` file example

> You can specify a different name, but must use this flag: `--sqlc-config [new_name.yaml]`
>
> Example: `pgxgen --sqlc-config sqlc-new.yaml`

```yaml
version: "2"
sql:
  - schema: "sql/migrations"
    queries: "sql/queries"
    engine: "postgresql"
    gen:
      go:
        sql_package: "pgx/v5"
        out: "internal/store"
        emit_prepared_queries: false
        emit_json_tags: true
        emit_exported_queries: false
        emit_db_tags: true
        emit_interface: true
        emit_exact_table_names: false
        emit_empty_slices: true
        emit_result_struct_pointers: true
        emit_params_struct_pointers: false
        emit_enum_valid_method: true
        emit_all_enum_values: true
```
