# pgxgen CLI Reference

## Installation

```bash
go install github.com/tkcrm/pgxgen/cmd/pgxgen@latest
```

## Global Flags

| Flag        | Default       | Description                |
| ----------- | ------------- | -------------------------- |
| `--config`  | `pgxgen.yaml` | Path to pgxgen config file |
| `--version` |               | Print version              |
| `--help`    |               | Show help                  |

## Commands

### generate

Generate code from schema. Default action generates everything.

```bash
pgxgen generate              # Generate all (crud + models + sqlc + constants)
pgxgen generate crud         # CRUD SQL queries only
pgxgen generate models       # Go models only
pgxgen generate constants    # Go constants only
pgxgen generate all          # Explicit "all" target
```

**Flags:**

| Flag        | Description                                   |
| ----------- | --------------------------------------------- |
| `--dry-run` | Preview changes without writing files         |
| `--debug`   | Show detailed timing for each generation step |

**Examples:**

```bash
pgxgen generate --dry-run                    # Preview all changes
pgxgen generate crud --debug                 # Generate CRUD with timing
pgxgen --config custom.yaml generate         # Use custom config path
```

### validate

Validate config and SQL schema. Useful for CI pipelines.

```bash
pgxgen validate
```

Checks:

- Config YAML structure and required fields
- `schema_dir` exists and contains parseable SQL files
- Tables referenced in config exist in SQL schema
- `primary_column` exists in its table

### watch

Watch schema files and regenerate on changes.

```bash
pgxgen watch
```

Monitors `.sql` files in all configured `schema_dir` paths. Triggers full regeneration on any change.

### init

Create `pgxgen.yaml` interactively.

```bash
pgxgen init
```

Prompts for: engine, schema directory, queries directory prefix, output directory prefix, models (y/n), sqlc (y/n). Fails if `pgxgen.yaml` already exists.

### example

Print an example v2 config with all features.

```bash
pgxgen example                       # PostgreSQL example (default)
pgxgen example --engine mysql        # MySQL example
pgxgen example --engine sqlite       # SQLite example
pgxgen example > pgxgen.yaml         # Save to file
```

### migrate

Migrate pgxgen.yaml from v1 to v2 format.

```bash
pgxgen migrate                                    # Print v2 config to stdout
pgxgen migrate --in-place                          # Overwrite (creates .v1.bak backup)
pgxgen migrate --in-place --sqlc-config sqlc.yaml  # Import sqlc.yaml settings
```

**Flags:**

| Flag            | Default     | Description                                          |
| --------------- | ----------- | ---------------------------------------------------- |
| `--in-place`    | `false`     | Overwrite pgxgen.yaml (creates .v1.bak backup)       |
| `--sqlc-config` | `sqlc.yaml` | Path to sqlc.yaml for extracting engine and settings |

### schema

Output consolidated DDL from all migration files. Reads all `.up.sql` files, replays CREATE/ALTER/DROP statements, and outputs the final schema state as a single DDL script. This command is standalone — it does not require a `pgxgen.yaml` config.

```bash
pgxgen schema sql/migrations/postgres                    # PostgreSQL (default engine)
pgxgen schema sql/migrations/sqlite -e sqlite            # SQLite
pgxgen schema sql/migrations/postgres > consolidated.sql # Redirect to file
pgxgen schema sql/migrations/001_init.up.sql             # Single file
```

**Arguments:**

| Argument | Description                          |
| -------- | ------------------------------------ |
| `<path>` | Path to migrations directory or file |

**Flags:**

| Flag             | Default      | Description                                 |
| ---------------- | ------------ | ------------------------------------------- |
| `--engine`, `-e` | `postgresql` | Database engine (postgresql, mysql, sqlite) |

**What is captured:**

- Tables with all columns (types, NOT NULL, defaults)
- PRIMARY KEY (inline and composite)
- FOREIGN KEY (inline REFERENCES and table-level) with ON DELETE/UPDATE actions
- UNIQUE and CHECK constraints
- CREATE INDEX (regular, unique, partial with WHERE, IF NOT EXISTS)
- CREATE EXTENSION (PostgreSQL)
- CREATE TYPE / ENUM (PostgreSQL)
- CREATE VIEW and materialized views
- COMMENT ON TABLE / COLUMN / VIEW (PostgreSQL)
- `ALTER TABLE ... RENAME TO` and `RENAME COLUMN` — applied to the final schema state (PostgreSQL and SQLite)
- Tables are topologically sorted by FK dependencies

### fmt

Format SQL files. Analyzes files, shows which need formatting, and asks for confirmation before writing. This command is standalone — it does not require a `pgxgen.yaml` config.

```bash
pgxgen fmt .                                        # Format all .sql files recursively
pgxgen fmt sql/migrations                           # Format directory recursively
pgxgen fmt sql/migrations/001_init.up.sql           # Format single file
pgxgen fmt . --check                                # Check only (exit 1 if unformatted, for CI)
pgxgen fmt . --dry-run                              # Process without saving (test formatting)
pgxgen fmt . --yes                                  # Skip confirmation prompt
pgxgen fmt . -e mysql                               # Use MySQL dialect
```

**Arguments:**

| Argument | Description                               |
| -------- | ----------------------------------------- |
| `<path>` | Path to SQL file or directory (recursive) |

**Flags:**

| Flag             | Default      | Description                                                      |
| ---------------- | ------------ | ---------------------------------------------------------------- |
| `--check`, `-c`  | `false`      | Check formatting without modifying files (exit 1 if unformatted) |
| `--dry-run`      | `false`      | Process files without saving (test formatting)                   |
| `--yes`, `-y`    | `false`      | Skip confirmation prompt                                         |
| `--engine`, `-e` | `postgresql` | SQL dialect (postgresql, mysql, sqlite)                          |

**Behavior:**

1. Finds all `.sql` files in `<path>` (recursively for directories)
2. Analyzes which files need formatting
3. Shows the list of files to format
4. Asks for confirmation (`y/N`) — skipped with `--yes` or `--check`
5. Writes formatted files

### update

Self-update pgxgen to the latest version.

```bash
pgxgen update
```

### version

Print version information.

```bash
pgxgen version
```

## Generation Pipeline

When `pgxgen generate` runs, the orchestrator executes these steps in order:

1. **Load config** — Parse `pgxgen.yaml` and validate
2. **Parse schema** — Read `.sql` files from `schema_dir`, build in-memory catalog (tables, columns, enums, types). Parsed once, reused by all generators.
3. **Generate CRUD SQL** — For each table with `crud.methods`, render engine-specific templates to `{queries_dir}/{table}_gen.sql`
4. **Generate custom queries** — Wrap `custom_queries` SQL in sqlc comment format
5. **Generate Go models** — Map SQL types to Go types, render structs with tags to `{models.output_dir}/{output_file_name}`
6. **Run sqlc** — Build `.pgxgen/sqlc.yaml`, invoke sqlc, post-process output (remove duplicate models.go, fix imports)
7. **Generate constants** — Render table/column name constants to `{output_dir}/constants_gen.go`
8. **Write results** — Persist all generated files to disk (or preview with `--dry-run`)

## Generated File Naming

| Generator   | Output file                       | Location                                     |
| ----------- | --------------------------------- | -------------------------------------------- |
| CRUD SQL    | `{table}_gen.sql`                 | `{queries_dir}/{table}/` or `{queries_dir}/` |
| Models      | `models_gen.go` (configurable)    | `{models.output_dir}/`                       |
| Constants   | `constants_gen.go`                | `{output_dir}/`                              |
| sqlc config | `sqlc.yaml`                       | `.pgxgen/` (auto-generated, not checked in)  |
| sqlc code   | `*.sql.go`, `db.go`, `querier.go` | `{output_dir}/`                              |

All generated Go files include the header: `// Code generated by pgxgen. DO NOT EDIT.`
