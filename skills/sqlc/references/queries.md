# sqlc Query Annotations Reference

## Table of Contents

- [Annotation syntax](#annotation-syntax)
- [Command types](#command-types)
- [Parameter syntax](#parameter-syntax)
- [Macros](#macros)
- [Examples by engine](#examples-by-engine)

## Annotation syntax

Every query must have a name comment directly above it:

```sql
-- name: QueryName :command
SELECT ...;
```

- **PostgreSQL/SQLite**: `-- name: QueryName :command`
- **MySQL**: `/* name: QueryName :command */`

The name becomes the Go method name. It must be a valid Go identifier (PascalCase recommended).

## Command types

| Command       | Returns               | Use case                        | All engines       |
| ------------- | --------------------- | ------------------------------- | ----------------- |
| `:one`        | `(Row, error)`        | SELECT single row               | Yes               |
| `:many`       | `([]Row, error)`      | SELECT multiple rows            | Yes               |
| `:exec`       | `error`               | INSERT/UPDATE/DELETE, no result | Yes               |
| `:execrows`   | `(int64, error)`      | Affected row count              | Yes               |
| `:execresult` | `(sql.Result, error)` | Full result with LastInsertId   | Yes               |
| `:execlastid` | `(int64, error)`      | Last inserted ID                | Yes               |
| `:copyfrom`   | `(int64, error)`      | Bulk insert via COPY            | PostgreSQL, MySQL |
| `:batchone`   | `BatchResults`        | Batch single-row query          | PostgreSQL (pgx)  |
| `:batchmany`  | `BatchResults`        | Batch multi-row query           | PostgreSQL (pgx)  |
| `:batchexec`  | `BatchResults`        | Batch exec                      | PostgreSQL (pgx)  |

### :one

Returns a single row. Uses `QueryRow` internally.

```sql
-- name: GetUser :one
SELECT id, name, email FROM users WHERE id = $1;
```

Generated: `func (q *Queries) GetUser(ctx context.Context, id int64) (User, error)`

### :many

Returns a slice of rows. Uses `Query` internally.

```sql
-- name: ListUsers :many
SELECT id, name, email FROM users ORDER BY name;
```

Generated: `func (q *Queries) ListUsers(ctx context.Context) ([]User, error)`

### :exec

Executes without returning data.

```sql
-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;
```

Generated: `func (q *Queries) DeleteUser(ctx context.Context, id int64) error`

### :execrows

Returns the number of affected rows.

```sql
-- name: SoftDeleteInactive :execrows
UPDATE users SET deleted_at = NOW() WHERE last_login < $1;
```

Generated: `func (q *Queries) SoftDeleteInactive(ctx context.Context, lastLogin time.Time) (int64, error)`

### :execresult

Returns `sql.Result` (for `LastInsertId()` and `RowsAffected()`).

```sql
-- name: InsertUser :execresult
INSERT INTO users (name, email) VALUES ($1, $2);
```

Generated: `func (q *Queries) InsertUser(ctx context.Context, arg InsertUserParams) (sql.Result, error)`

### :execlastid

Returns the last inserted ID directly.

```sql
-- name: InsertUser :execlastid
INSERT INTO users (name, email) VALUES ($1, $2);
```

Generated: `func (q *Queries) InsertUser(ctx context.Context, arg InsertUserParams) (int64, error)`

### :copyfrom

Bulk insert using PostgreSQL COPY or MySQL LOAD DATA. Generates a method accepting a slice.

```sql
-- name: BulkInsertUsers :copyfrom
INSERT INTO users (name, email) VALUES ($1, $2);
```

Generated: `func (q *Queries) BulkInsertUsers(ctx context.Context, arg []BulkInsertUsersParams) (int64, error)`

Requirements:

- PostgreSQL: requires `pgx/v4`, `pgx/v5`, or `database/sql` with `sql_driver` set
- MySQL: requires `database/sql` with `sql_driver: "github.com/go-sql-driver/mysql"`
- Only INSERT statements supported

### :batchone, :batchmany, :batchexec

Batch operations use pgx's batch query pipeline. Only available with `pgx/v4` or `pgx/v5`.

```sql
-- name: GetUserBatch :batchone
SELECT id, name, email FROM users WHERE id = $1;

-- name: ListUsersByCity :batchmany
SELECT id, name FROM users WHERE city = $1;

-- name: DeleteUserBatch :batchexec
DELETE FROM users WHERE id = $1;
```

Generated batch types have:

- Constructor that queues queries
- `Query`/`QueryRow`/`Exec` method with callback `func(int, T, error)` or `func(int, error)`
- `Close() error` to release resources

## Parameter syntax

### PostgreSQL

Uses numbered parameters: `$1`, `$2`, `$3`, etc.

```sql
-- name: CreateUser :one
INSERT INTO users (name, email, role)
VALUES ($1, $2, $3)
RETURNING *;
```

### MySQL

Uses `?` placeholders:

```sql
/* name: CreateUser :execresult */
INSERT INTO users (name, email, role) VALUES (?, ?, ?);
```

### SQLite

Uses `?` placeholders (same as MySQL):

```sql
-- name: CreateUser :execlastid
INSERT INTO users (name, email) VALUES (?, ?);
```

## Macros

### sqlc.arg(name)

Name a parameter explicitly. Useful when parameter order doesn't match struct field order,
or when you want descriptive names.

```sql
-- name: UpdateUserEmail :exec
UPDATE users
SET email = sqlc.arg('new_email')
WHERE id = sqlc.arg('user_id');
```

Generated params struct:

```go
type UpdateUserEmailParams struct {
    NewEmail string
    UserID   int64
}
```

### @name (PostgreSQL only)

Shorthand for `sqlc.arg(name)`:

```sql
-- name: UpdateUserEmail :exec
UPDATE users SET email = @new_email WHERE id = @user_id;
```

Equivalent to `sqlc.arg('new_email')` and `sqlc.arg('user_id')`.

### sqlc.narg(name)

Like `sqlc.arg()` but marks the parameter as nullable. The Go type will be a pointer
or `sql.Null*` type.

```sql
-- name: UpdateUserBio :exec
UPDATE users SET bio = sqlc.narg('bio') WHERE id = sqlc.arg('user_id');
```

Generated:

```go
type UpdateUserBioParams struct {
    Bio    *string  // nullable because of sqlc.narg
    UserID int64
}
```

### sqlc.embed(table)

Embed a table's struct in the result. Useful for JOINs to avoid flat result structs.

```sql
-- name: GetUserWithPosts :many
SELECT sqlc.embed(users), sqlc.embed(posts)
FROM users
JOIN posts ON posts.author_id = users.id
WHERE users.id = $1;
```

Generated:

```go
type GetUserWithPostsRow struct {
    User User
    Post Post
}
```

Without `sqlc.embed`, you'd get a flat struct with all columns from both tables.

### sqlc.slice(name)

Dynamic IN clause for MySQL and SQLite. Generates code that expands the slice
into the correct number of placeholders at runtime.

```sql
/* name: GetUsersByIDs :many */
SELECT * FROM users WHERE id IN (sqlc.slice('ids'));
```

Generated:

```go
func (q *Queries) GetUsersByIDs(ctx context.Context, ids []int64) ([]User, error)
```

**Not needed for PostgreSQL** — use `ANY($1::int[])` with array types instead:

```sql
-- name: GetUsersByIDs :many
SELECT * FROM users WHERE id = ANY($1::int[]);
```

## Examples by engine

### PostgreSQL with pgx/v5

```sql
-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (name, email)
VALUES ($1, $2)
RETURNING *;

-- name: SearchUsers :many
SELECT * FROM users
WHERE name ILIKE '%' || $1 || '%'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpsertUser :one
INSERT INTO users (email, name)
VALUES ($1, $2)
ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name
RETURNING *;
```

### MySQL

```sql
/* name: GetUser :one */
SELECT * FROM users WHERE id = ?;

/* name: CreateUser :execresult */
INSERT INTO users (name, email) VALUES (?, ?);

/* name: SearchUsers :many */
SELECT * FROM users
WHERE name LIKE CONCAT('%', ?, '%')
ORDER BY created_at DESC
LIMIT ? OFFSET ?;
```

### SQLite

```sql
-- name: GetUser :one
SELECT * FROM users WHERE id = ?;

-- name: CreateUser :execlastid
INSERT INTO users (name, email) VALUES (?, ?);

-- name: ListUsers :many
SELECT * FROM users ORDER BY name LIMIT ? OFFSET ?;
```
