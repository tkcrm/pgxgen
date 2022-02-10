# pgxgen

pgxgen use [`sqlc`](https://github.com/kyleconroy/sqlc) tool with additional improvements.

- Instead null types like `sql.NullString` used nil type `*string` by default
- Generate CRUD for existing tables in postgresql database
- Json tags: Omit empty and hide
- Use Sqlc only for generating models
- Update generated models with additinal parameters: add / update fields and tags
- Generate models for [`Mobx Keystone`](https://github.com/xaviergonz/mobx-keystone)

## Install

### Requirements

- `Go 1.17+`

```bash
go install github.com/tkcrm/pgxgen/cmd/pgxgen@latest
```

## Usage

### Configure `pgxgen`

At root of your project create a `pgxgen.yaml`. Example of configuration below.

```yaml
version: 1
# Result SQL file name; default: crud_queries.sql
# Will save to `queries` path from `sqlc.yaml` config
output_crud_sql_file_name: "crud_queries.sql"
# Generate models parameters. Not required
gen_models:
  # default: false
  - delete_sqlc_data: true
    # required
    models_output_dir: "internal/model"
    # default: models.go
    models_output_filename: "models.go"
    # default: last item in models_output_dir
    models_package_name: "model"
    models_imports:
      - "github.com/uptrace/bun"
    # Use uint64 instead int64 for all fields ends with ID
    use_uint_for_ids: true
    use_uint_for_ids_exceptions:
      - struct_name: "users"
        field_names:
          - OrganizationID
          - UserID
    add_fields:
      - struct_name: "users"
        # default: start
        # available values: start, end, after FieldName
        position: "start"
        field_name: "bun.BaseModel"
        type: ""
        tags:
          - name: "bun"
            value: "table:users,alias:u"
    update_fields:
      - struct_name: "users"
        field_name: "Password"
        new_parameters:
          name: "Password"
          type: "string"
          # default: false
          match_with_current_tags: true
          tags:
            - name: "json"
              value: "-"
    delete_fields:
      - struct_name: "users"
        field_names:
          - CreatedAt
          - UpdatedAt
    external_models:
      keystone:
        # required
        output_dir: "frontend/src/stores/models"
        # default: models.ts
        output_file_name: "models.ts"
        # default: empty
        decorator_model_name_prefix: "frontend/"
        # set method .withSetter() for all fields
        with_setter: true
        export_model_suffix: "Model"
        # prettier code. nodejs must be installed on your pc
        prettier_code: true
        # sort output models
        # you can specify only those structures that need to be generated
        # in the first place and omit all the rest
        sort: "UserRole,Users"
        # params are currently unavailable
        params:
          - struct_name: "users"
            field_name: "organization"
            field_params:
              - with_setter: false
# Update json tag. Not required
json_tags:
  # List of struct fields
  # Convert: `json:"field_name"` => `json:"field_name,omitempty"`
  omitempty:
    - created_at
  # List of struct fields
  # Convert: `json:"field_name"` => `json:"-"`
  hide:
    - password
crud_params:
  # Limit and offset for `Find` method
  limit:
    # List of tables or asterisk (*)
    - "*"
  # Order by for `Find` method
  order_by:
    - by: id
      order: desc
      tables:
        # List of tables or asterisk (*)
        - "*"
  where:
    # g - get
    # f - find
    # u - update
    # d - delete
    # t - total
    # available asterisk (*) for all methods (gfudt) except create
    - methods: "gfudt"
      # List of tables or asterisk (*)
      tables:
        - users
      params:
        - organization_id
```

### Generate `CRUD` queries for existing tables

```bash
pgxgen gencrud -c=postgres://DB_USER:DB_PASSWD@DB_HOST:DB_PORT/DB_NAME?sslmode=disable
```

### Generate models based on sqlc models

```bash
pgxgen genmodels
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
    schema: "sql/migrations"
    queries: "sql/queries"
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

### Generate `sqlc`

```bash
pgxgen sqlc generate
```

## Roadmap

- Generate models without sqlc
- Generate `.proto` files with CRUD services
