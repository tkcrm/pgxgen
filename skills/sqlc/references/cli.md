# sqlc CLI Reference

## Commands

### sqlc init

Create an empty `sqlc.yaml` configuration file.

```bash
sqlc init              # Creates v2 config (default)
sqlc init --v1         # Creates v1 config (legacy)
sqlc init --v2         # Creates v2 config (explicit)
```

### sqlc generate

Generate Go code from SQL queries and schema.

```bash
sqlc generate                  # Use default config file
sqlc generate -f sqlc.yaml     # Specify config file
```

Reads config, parses schema and queries, generates code in the configured output directories.

### sqlc compile

Parse and type-check SQL without generating code. Useful for CI validation.

```bash
sqlc compile
sqlc compile -f sqlc.yaml
```

Alias: `sqlc check`

Catches:

- SQL syntax errors
- Unknown tables/columns
- Type mismatches
- Invalid annotations

### sqlc vet

Run lint rules against queries.

```bash
sqlc vet
sqlc vet -f sqlc.yaml
```

Requires rules configured in `sqlc.yaml`. See [vet.md](vet.md) for rule configuration.
Exits non-zero if any rule fails.

### sqlc diff

Compare generated code against existing files. Exits non-zero if they differ.

```bash
sqlc diff
sqlc diff -f sqlc.yaml
```

Ideal for CI to ensure generated code is committed:

```yaml
# In CI pipeline
- run: sqlc diff
```

### sqlc parse

Parse SQL and output the AST as JSON.

```bash
sqlc parse schema.sql                    # Auto-detect dialect
sqlc parse --dialect postgresql file.sql # Explicit dialect
echo "SELECT 1" | sqlc parse --dialect mysql  # From stdin
```

Supported dialects: `postgresql`, `mysql`, `sqlite`, `clickhouse`

### sqlc verify

Verify that generated code matches the current schema and queries (requires sqlc Cloud).

```bash
sqlc verify
```

### sqlc createdb

Create a database (requires sqlc Cloud with managed databases).

```bash
sqlc createdb
```

### sqlc push

Push schema and queries to sqlc Cloud.

```bash
sqlc push
sqlc push --dry-run    # Preview without pushing
```

### sqlc version

Print the sqlc version.

```bash
sqlc version
```

## Global flags

| Flag                | Description                                     |
| ------------------- | ----------------------------------------------- |
| `-f, --file <path>` | Specify config file path (default: `sqlc.yaml`) |
| `--no-remote`       | Disable remote execution                        |
| `--remote`          | Enable remote execution                         |

## Common CI workflows

### Basic validation

```bash
sqlc compile && sqlc diff
```

### Full validation with vet

```bash
sqlc compile && sqlc vet && sqlc diff
```

### Generate and commit check

```bash
sqlc generate
git diff --exit-code -- internal/db/
```
