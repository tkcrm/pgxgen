# CRUD Method Configuration

## Method Types

| Method         | SQL Operation                 | sqlc Type                          | Description                 |
| -------------- | ----------------------------- | ---------------------------------- | --------------------------- |
| `create`       | INSERT                        | `:one` (with returning) or `:exec` | Insert a row                |
| `update`       | UPDATE                        | `:one` (with returning) or `:exec` | Update a row by primary key |
| `delete`       | DELETE                        | `:exec`                            | Delete a row by primary key |
| `get`          | SELECT ... LIMIT 1            | `:one`                             | Get one row by primary key  |
| `find`         | SELECT with WHERE/ORDER/LIMIT | `:many`                            | List rows with filtering    |
| `total`        | SELECT count(1)               | `:one`                             | Count rows                  |
| `exists`       | SELECT EXISTS                 | `:one`                             | Check if a row exists       |
| `batch_create` | INSERT (multi-row)            | `:copyfrom`                        | Batch insert rows           |

## Common Method Options

All methods support:

```yaml
name: CustomMethodName # Override generated method name
```

## create

```yaml
create:
  name: CreateUser # Override method name
  skip_columns: [id, updated_at] # Columns to exclude from INSERT
  column_values: # Fixed values (not parameterized)
    created_at: "now()"
    status: "'active'" # String literals need inner quotes
  returning: "*" # PostgreSQL only. RETURNING clause
```

**Generated SQL (PostgreSQL):**

```sql
-- name: CreateUser :one
INSERT INTO users (name, email, created_at)
VALUES ($1, $2, now())
RETURNING *;
```

**Generated SQL (MySQL):**

```sql
-- name: CreateUser :exec
INSERT INTO users (name, email, created_at)
VALUES (?, ?, NOW());
```

## update

```yaml
update:
  name: UpdateUser
  skip_columns: [id, created_at] # Columns to exclude from SET
  column_values:
    updated_at: "now()"
  returning: "*" # PostgreSQL only
```

**Generated SQL:**

```sql
-- name: UpdateUser :one
UPDATE users SET name = $1, email = $2, updated_at = now()
WHERE id = $3
RETURNING *;
```

## delete

```yaml
delete:
  name: DeleteUser
```

**Without soft delete:**

```sql
-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;
```

**With soft delete** (`soft_delete.column: deleted_at`):

```sql
-- name: DeleteUser :exec
UPDATE users SET deleted_at = now() WHERE id = $1;
```

## get

```yaml
get:
  name: GetByID
  where: # Additional WHERE conditions
    tenant_id: {} # Adds: AND tenant_id = $2
```

**Generated SQL:**

```sql
-- name: GetByID :one
SELECT * FROM users WHERE id = $1 LIMIT 1;
```

**With soft delete** auto-adds `AND deleted_at IS NULL`:

```sql
-- name: GetByID :one
SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL LIMIT 1;
```

## find

```yaml
find:
  name: FindUsers
  where: # WHERE conditions (all parameterized)
    status: {} # status = $1
    role:
      operator: "!=" # role != $2
  where_additional: # Raw WHERE clauses
    - "created_at > $3"
  order:
    by: created_at
    direction: DESC # ASC or DESC (default: DESC)
  limit: true # Adds LIMIT $N OFFSET $M parameters
```

**Generated SQL:**

```sql
-- name: FindUsers :many
SELECT * FROM users
WHERE status = $1 AND role != $2 AND created_at > $3
ORDER BY created_at DESC
LIMIT $4 OFFSET $5;
```

**With soft delete** auto-adds `deleted_at IS NULL` to WHERE.

### Where operators

```yaml
where:
  email: {}                    # email = $N (default operator)
  email:
    operator: "="              # Explicit equals
  age:
    operator: ">="             # age >= $N
  role:
    operator: "!="             # role != $N
  name:
    operator: LIKE             # name LIKE $N
  status:
    value: "'active'"          # Fixed value, not parameterized: status = 'active'
```

## total

```yaml
total:
  name: CountUsers
  where:
    status: {} # Optional WHERE
```

**Generated SQL:**

```sql
-- name: CountUsers :one
SELECT count(1) FROM users WHERE status = $1;
```

## exists

```yaml
exists:
  name: UserExistsByEmail
  where:
    email: {}
```

**Generated SQL (PostgreSQL):**

```sql
-- name: UserExistsByEmail :one
SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)::boolean;
```

**Generated SQL (MySQL/SQLite):**

```sql
-- name: UserExistsByEmail :one
SELECT EXISTS (SELECT 1 FROM users WHERE email = ?);
```

## batch_create

```yaml
batch_create:
  name: BatchCreateUsers
  skip_columns: [id, created_at]
```

**Generated SQL (PostgreSQL — uses COPY protocol):**

```sql
-- name: BatchCreateUsers :copyfrom
INSERT INTO users (name, email) VALUES ($1, $2);
```

**Generated SQL (MySQL — uses multi-value INSERT):**

```sql
-- name: BatchCreateUsers :copyfrom
INSERT INTO users (name, email) VALUES (?, ?);
```

## Default Methods Inheritance

Methods defined in `defaults.crud.methods` apply to all tables. Per-table `crud.methods` override specific methods. Only methods explicitly listed in a table's `crud.methods` are generated for that table.

```yaml
defaults:
  crud:
    methods:
      create: # Default create config for all tables
        skip_columns: [id, updated_at]
        returning: "*"

tables:
  users:
    crud:
      methods:
        create: # Overrides default create for users
          skip_columns: [id, updated_at, role]
          returning: "*"
        get: {} # Uses no defaults — just empty config
```

## Method Naming

With `exclude_table_name: false` (default):

- create → `Create{Table}` (e.g., `CreateUser`)
- get → `Get{Table}ByID` (e.g., `GetUserByID`)

With `exclude_table_name: true`:

- create → `Create`
- get → `GetByID`

The `name` field always overrides the generated name.
