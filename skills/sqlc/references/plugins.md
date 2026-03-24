# sqlc Plugins Reference

## Overview

Plugins extend sqlc's code generation beyond the built-in Go generator. Two plugin types:

- **WASM** — Sandboxed WebAssembly modules. Recommended for security.
- **Process** — Local executables. Useful for development and custom tooling.

## Configuration

Plugins are defined at the top level and referenced in SQL set `codegen` sections:

```yaml
version: "2"
plugins:
  - name: "my-plugin"
    wasm:
      url: "https://github.com/.../plugin.wasm"
      sha256: "checksum..."

sql:
  - schema: "schema.sql"
    queries: "queries/"
    engine: "postgresql"
    codegen:
      - out: "gen/"
        plugin: "my-plugin"
        options:
          key: "value"
```

## WASM plugins

Fully sandboxed — no filesystem, network, or environment access except explicitly whitelisted vars.

```yaml
plugins:
  - name: "sqlc-gen-typescript"
    wasm:
      url: "https://github.com/sqlc-dev/sqlc-gen-typescript/releases/download/v0.1.0/sqlc-gen-typescript.wasm"
      sha256: "abc123def456..."
    env:
      - NODE_ENV # Whitelist specific env vars
```

- `url` — Download URL for the `.wasm` file
- `sha256` — Required checksum for integrity verification
- `env` — Optional list of environment variable names to pass to the plugin

## Process plugins

Run a local command. The command receives a `CodeGenRequest` on stdin and writes `CodeGenResponse` to stdout.

```yaml
plugins:
  - name: "sqlc-gen-json"
    process:
      cmd: "sqlc-gen-json" # Must be in PATH or absolute path
      format: "json" # "json" or "protobuf" (default: "protobuf")
    env:
      - PATH
      - HOME
```

- `cmd` — Command to execute
- `format` — Serialization format for stdin/stdout communication
- `env` — Environment variables to pass through

**Security note**: Process plugins run arbitrary code with the same permissions as the sqlc process. Only use trusted plugins.

## codegen section

Reference plugins in SQL set codegen:

```yaml
sql:
  - schema: "schema.sql"
    queries: "queries/"
    engine: "postgresql"
    gen:
      go: # Built-in Go generation (optional)
        package: "db"
        out: "internal/db"
    codegen:
      - out: "gen/typescript"
        plugin: "sqlc-gen-typescript"
        options: # Plugin-specific options (passed as JSON)
          runtime: "node"
          driver: "pg"
      - out: "gen/json"
        plugin: "sqlc-gen-json"
        options:
          indent: "  "
          filename: "queries.json"
```

- `out` — Output directory for generated files
- `plugin` — Must match a plugin name from top-level `plugins`
- `options` — Arbitrary key-value pairs passed to the plugin

A single SQL set can have both `gen.go` and multiple `codegen` entries.

## Environment variables

- `SQLC_VERSION` — Always available in all plugins (built-in)
- Additional vars must be explicitly listed in the plugin's `env` array
- WASM plugins: only listed vars are accessible (strict sandbox)
- Process plugins: only listed vars are passed to the subprocess

## Built-in plugin: sqlc-gen-json

Serializes the `CodeGenRequest` protobuf to JSON. Useful for debugging and building custom tooling.

```yaml
plugins:
  - name: "jsonb"
    process:
      cmd: "sqlc-gen-json"
      format: "json"

sql:
  - schema: "schema.sql"
    queries: "queries/"
    engine: "postgresql"
    codegen:
      - out: "gen"
        plugin: "jsonb"
        options:
          indent: "  "
          filename: "codegen_request.json"
```

Build it from the sqlc repo: `go build -o ~/go/bin/sqlc-gen-json ./cmd/sqlc-gen-json`

## Writing custom plugins

A plugin is any program that:

1. Reads a `CodeGenRequest` from stdin (JSON or protobuf)
2. Writes a `CodeGenResponse` to stdout

The protobuf definitions are in the sqlc repository under `protos/`. Use `sqlc-gen-json` output to understand the request structure.
