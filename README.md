# pgxgen

pgxgen use [`sqlc`](https://github.com/sqlc-dev/sqlc) tool with additional improvements.

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

- `Go 1.23+`

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
   crud      Generate crud sql's
   gomodels  Generate golang models based on existed structs
   keystone  Generate mobx keystone models
   ts        Generate types for typescript, based on go structs
   sqlc      Generate sqlc code
   update    Update pgxgen to the latest version
   version   Print the version
   help, h   Shows a list of commands or help for one command

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

# modification of existing models. not required
gen_models:
  - # path to a specific file
    input_file_path: "internal/store/models.go"
    # input dir. will process all files with extension `.go`
    input_dir: "internal/store"
    # delete specific file or all files in dir. default: false
    delete_original_files: false
    # output dir. required
    output_dir: "internal/models"
    # output file name. required
    output_file_name: "models_gen.go"
    # default: last item in output_dir
    package_name: "model"
    # additional imports
    imports:
      - "github.com/uptrace/bun"
    # Use uint64 instead int64 for all fields ends with ID
    use_uint_for_ids: true
    use_uint_for_ids_exceptions:
      - struct_name: "User"
        field_names:
          - OrganizationID
          - UserID
    # add new field to struct
    add_fields:
      - struct_name: "User"
        # default: start
        # available values: start, end, after FieldName
        position: "start"
        field_name: "bun.BaseModel"
        type: ""
        tags:
          - name: "bun"
            value: "table:users,alias:u"
    # update fields for all structs
    update_all_struct_fields:
      # update by field name
      by_field:
        - field_name: "ID"
          new_field_name: "bun.NullTime"
          new_type: "int"
          match_with_current_tags: true
          tags:
            - name: "json"
              value: "-"
      # update by field type
      by_type:
        - type: "*time.Time"
          new_type: "bun.NullTime"
          match_with_current_tags: true
          tags:
            - name: "json"
              value: "-"
    # update fields in specific struct
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
    # delete specific field in struct
    delete_fields:
      - struct_name: "users"
        field_names:
          - CreatedAt
          - UpdatedAt
    rename:
      oldName: newName
    # exclude structs from result list
    exclude_structs:
      - struct_name: "User"
    # only the listed structures will be used
    include_structs:
      - struct_name: "User"

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
```

### Install `@tkcrm/ui` in your frontend

If you generate mobx keystone models install `@tkcrm/ui` in your frontend project

```bash
npm i @tkcrm/ui --save-dev
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
