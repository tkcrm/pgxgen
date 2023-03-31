# pgxgen

pgxgen use [`sqlc`](https://github.com/kyleconroy/sqlc) tool with additional improvements.

- Instead null types like `sql.NullString` used nil type `*string` by default
- Generate CRUD for existing tables in postgresql database
- Json tags: Omit empty and hide
- Use Sqlc only for generating models
- Update generated models with additinal parameters: add / update fields and tags
- Generate models for [`Mobx Keystone`](https://github.com/xaviergonz/mobx-keystone)
- Generate typescript code based on go structs

> You can use [this repository](https://github.com/sxwebdev/pgxgen-example) which explains how to use `pgxgen` tool in your project

## Install

### Requirements

- `Go 1.19+`

```bash
go install github.com/tkcrm/pgxgen/cmd/pgxgen@latest
```

## Usage

### Help

Print all available commands

```bash
pgxgen help
```

### Version

```bash
# Print current version
pgxgen version

# Check for new version
pgxgen check-version
```

### Configure `pgxgen`

At root of your project create a `pgxgen.yaml`. Example of configuration below.

> You can specify a different name, but must use this flag: `--pgxgen-config [new_name.yaml]`
>
> Example: `pgxgen --pgxgen-config pgxgen-new.yaml`

```yaml
version: 1
disable_auto_replace_sqlc_nullable_types: false
# move sqlc models to another package and directory
sqlc_move_models:
  # required
  output_dir: "internal/models"
  # default: models.go
  output_filename: "models.go"
  # new package name. by default based on `output_dir`
  package_name: models
  # required. full path to new models directory
  package_path: github.com/company/project/internal/models
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
    update_all_struct_fields:
      by_field:
        - field_name: "ID"
          new_field_name: "bun.NullTime"
          new_type: "int"
          match_with_current_tags: true
          tags:
            - name: "json"
              value: "-"
      by_type:
        - type: "*time.Time"
          new_type: "bun.NullTime"
          match_with_current_tags: true
          tags:
            - name: "json"
              value: "-"
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
gen_keystone_models:
  - input_file_path: "internal/models/models_gen.go"
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
    # skip models
    skip_models:
      - NullUserRole
    # params are currently unavailable
    params:
      - struct_name: "users"
        field_name: "organization"
        field_params:
          - with_setter: false
gen_typescript_from_structs:
  - path: "pb"
    output_dir: "frontend/src/stores/models"
    output_file_name: "requests.d.ts"
    prettier_code: true
    export_type_prefix: "Store"
    export_type_suffix: "Gen"
    include_struct_names_regexp:
      - "^\\w*Request$"
      - "^\\w*Response$"
    exclude_struct_names_regexp:
      - "GetUserRequest"
      - "GetOrganizationRequest"
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
          returning: "*"
        update:
          skip_columns:
            - id
            - created_at
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
go_constants:
  tables:
    users:
      output_dir: internal/store/users/repo_users
```

### Generate `CRUD` queries for existing tables

```bash
pgxgen crud
```

### Generate models based on sqlc models

```bash
pgxgen gomodels
```

### Generate keystone models based on go models

```bash
pgxgen keystone
```

#### Install `@tkcrm/ui` in your frontend

If you generate mobx keystone models install `@tkcrm/ui` in your frontend project

```bash
npm i @tkcrm/ui --save-dev
```

### Generate typescript types based on go structs

```bash
pgxgen ts
```

### Configure `sqlc`

At root of your project create a `sqlc.yaml` file with the configuration described below.

> Configuration available [here](https://docs.sqlc.dev/en/stable/reference/config.html)

#### Configuration `sqlc.yaml` file example

> You can specify a different name, but must use this flag: `--sqlc-config [new_name.yaml]`
>
> Example: `pgxgen --sqlc-config sqlc-new.yaml`

```yaml
version: 2
sql:
  - schema: "sql/migrations"
    queries: "sql/queries"
    engine: "postgresql"
    gen:
      go:
        sql_package: "pgx/v4"
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

### Generate `sqlc`

```bash
pgxgen sqlc generate
```

## Roadmap

- Generate models without sqlc
- Generate `.proto` files with CRUD services
