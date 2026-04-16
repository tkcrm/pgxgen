package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/tkcrm/pgxgen/internal/codegen"
	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/ddlgen"
	"github.com/tkcrm/pgxgen/internal/sqlfmt"
	"github.com/tkcrm/pgxgen/internal/sqlfmt/formatters"
	"github.com/tkcrm/pgxgen/internal/sqlfmt/lexer"
	"github.com/tkcrm/pgxgen/internal/sqlparser"
	"github.com/tkcrm/pgxgen/internal/updater"
	"github.com/tkcrm/pgxgen/internal/watcher"
	"github.com/tkcrm/pgxgen/pkg/logger"
	"github.com/urfave/cli/v3"
)

// NewApp creates the pgxgen CLI application.
func NewApp(version string) *cli.Command {
	l := logger.New()

	return &cli.Command{
		Name:    "pgxgen",
		Version: fmt.Sprintf("\nrelease: %s\ngo version: %s", version, runtime.Version()),
		Usage:   "Generate Go models and DB CRUD from SQL schema",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Usage: "Path to pgxgen.yaml",
				Value: "pgxgen.yaml",
			},
		},
		Commands: []*cli.Command{
			newGenerateCmd(l),
			newSchemaCmd(l),
			newMigrateCmd(l),
			newValidateCmd(l),
			newInitCmd(l),
			newWatchCmd(l),
			newExampleCmd(l),
			newFmtCmd(l),
			newVersionCmd(version),
			newUpdateCmd(l, version),
		},
	}
}

// loadV2Config loads the v2 config from the CLI context.
func loadV2Config(cmd *cli.Command) (*config.V2Config, error) {
	configPath := cmd.String("config")
	cfg, err := config.LoadV2Config(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config error: %w", err)
	}
	return cfg, nil
}

func runGenerate(ctx context.Context, cmd *cli.Command, l logger.Logger, targets []string) error {
	configPath := cmd.String("config")
	cfg, err := loadV2Config(cmd)
	if err != nil {
		return err
	}

	dryRun := cmd.Bool("dry-run")
	debug := cmd.Bool("debug")
	orch := codegen.NewOrchestrator(l, cfg, configPath)
	_, err = orch.Generate(ctx, codegen.GenerateOpts{
		DryRun:  dryRun,
		Debug:   debug,
		Targets: targets,
	})
	return err
}

func newGenerateCmd(l logger.Logger) *cli.Command {
	dryRunFlag := &cli.BoolFlag{
		Name:  "dry-run",
		Usage: "Preview changes without writing files",
	}
	debugFlag := &cli.BoolFlag{
		Name:  "debug",
		Usage: "Show detailed timing for each generation step",
	}
	commonFlags := []cli.Flag{dryRunFlag, debugFlag}

	return &cli.Command{
		Name:  "generate",
		Usage: "Generate code from schema",
		Flags: commonFlags,
		Commands: []*cli.Command{
			{
				Name:  "crud",
				Usage: "Generate CRUD SQL queries",
				Flags: commonFlags,
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return runGenerate(ctx, cmd, l, []string{"crud"})
				},
			},
			{
				Name:  "models",
				Usage: "Generate Go models from SQL schema",
				Flags: commonFlags,
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return runGenerate(ctx, cmd, l, []string{"models"})
				},
			},
			{
				Name:  "constants",
				Usage: "Generate Go constants for table/column names",
				Flags: commonFlags,
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return runGenerate(ctx, cmd, l, []string{"constants"})
				},
			},
			{
				Name:  "all",
				Usage: "Generate everything (crud + models + sqlc + constants)",
				Flags: commonFlags,
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return runGenerate(ctx, cmd, l, nil)
				},
			},
		},
		// Default action: generate all
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runGenerate(ctx, cmd, l, nil)
		},
	}
}

func newSchemaCmd(_ logger.Logger) *cli.Command {
	return &cli.Command{
		Name:      "schema",
		Usage:     "Output consolidated DDL from all migrations",
		ArgsUsage: "<path>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "engine",
				Aliases: []string{"e"},
				Usage:   "Database engine (postgresql, mysql, sqlite)",
				Value:   "postgresql",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return fmt.Errorf("path argument is required\nusage: pgxgen schema <path> [--engine <engine>]")
			}
			dir := cmd.Args().First()
			engine := cmd.String("engine")

			files, err := sqlparser.ResolveSchemaFiles(dir)
			if err != nil {
				return fmt.Errorf("resolve schema files: %w", err)
			}

			parser, err := sqlparser.NewParser(engine)
			if err != nil {
				return err
			}

			cat, err := parser.ParseSchema(files)
			if err != nil {
				return fmt.Errorf("parse schema: %w", err)
			}

			gen, err := ddlgen.New(engine)
			if err != nil {
				return err
			}

			ddl, err := gen.Generate(cat)
			if err != nil {
				return fmt.Errorf("generate DDL: %w", err)
			}

			fmt.Print(ddl)
			return nil
		},
	}
}

func newMigrateCmd(l logger.Logger) *cli.Command {
	return &cli.Command{
		Name:  "migrate",
		Usage: "Migrate pgxgen.yaml from v1 to v2",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "in-place",
				Usage: "Overwrite pgxgen.yaml (creates .v1.bak backup)",
			},
			&cli.StringFlag{
				Name:  "sqlc-config",
				Usage: "Path to sqlc.yaml (for extracting engine and settings)",
				Value: "sqlc.yaml",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			pgxgenPath := cmd.String("config")
			sqlcPath := cmd.String("sqlc-config")

			v2yaml, err := config.MigrateV1ToV2(pgxgenPath, sqlcPath)
			if err != nil {
				return fmt.Errorf("migration failed: %w", err)
			}

			if cmd.Bool("in-place") {
				// Backup v1
				backupPath := pgxgenPath + ".v1.bak"
				data, err := os.ReadFile(pgxgenPath)
				if err != nil {
					return fmt.Errorf("read original config: %w", err)
				}
				if err := os.WriteFile(backupPath, data, 0o644); err != nil {
					return fmt.Errorf("write backup: %w", err)
				}
				l.Infof("backup saved to %s", backupPath)

				if err := os.WriteFile(pgxgenPath, v2yaml, 0o644); err != nil {
					return fmt.Errorf("write v2 config: %w", err)
				}
				l.Infof("v2 config written to %s", pgxgenPath)
			} else {
				fmt.Print(string(v2yaml))
			}

			l.Info("Migration complete. You can now delete your sqlc.yaml — pgxgen v2 generates it automatically.")
			return nil
		},
	}
}

func newValidateCmd(l logger.Logger) *cli.Command {
	return &cli.Command{
		Name:  "validate",
		Usage: "Validate config and SQL schema",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg, err := loadV2Config(cmd)
			if err != nil {
				return err
			}

			// Validate with go-playground/validator
			validate := validator.New()
			if err := validate.Struct(cfg); err != nil {
				return fmt.Errorf("config validation failed:\n%w", err)
			}

			configDir := filepath.Dir(cmd.String("config"))

			// Validate each schema
			for _, schema := range cfg.Schemas {
				schemaDir := schema.SchemaDir
				if !filepath.IsAbs(schemaDir) {
					schemaDir = filepath.Join(configDir, schemaDir)
				}

				// Check schema_dir exists
				if _, err := os.Stat(schemaDir); os.IsNotExist(err) {
					return fmt.Errorf("schema %q: schema_dir %q does not exist", schema.Name, schema.SchemaDir)
				}

				// Try parsing schema
				parser, err := sqlparser.NewParser(schema.Engine)
				if err != nil {
					return fmt.Errorf("schema %q: %w", schema.Name, err)
				}

				files, err := sqlparser.ResolveSchemaFiles(schemaDir)
				if err != nil {
					return fmt.Errorf("schema %q: resolve files: %w", schema.Name, err)
				}

				cat, err := parser.ParseSchema(files)
				if err != nil {
					return fmt.Errorf("schema %q: parse error: %w", schema.Name, err)
				}

				// Validate tables exist in schema
				for tableName, tableConfig := range schema.Tables {
					found := false
					for _, s := range cat.Schemas {
						for _, t := range s.Tables {
							if t.Name == tableName {
								found = true
								// Validate primary_column exists
								if tableConfig.PrimaryColumn != "" {
									colFound := false
									for _, c := range t.Columns {
										if c.Name == tableConfig.PrimaryColumn {
											colFound = true
											break
										}
									}
									if !colFound {
										return fmt.Errorf("schema %q: table %q: primary_column %q not found", schema.Name, tableName, tableConfig.PrimaryColumn)
									}
								}
								break
							}
						}
						if found {
							break
						}
					}
					if !found {
						return fmt.Errorf("schema %q: table %q not found in SQL schema", schema.Name, tableName)
					}
				}

				l.Infof("schema %q: valid (%d tables, %d SQL files)", schema.Name, len(schema.Tables), len(files))
			}

			l.Info("config is valid")
			return nil
		},
	}
}

func newInitCmd(l logger.Logger) *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Create pgxgen.yaml interactively",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			configPath := cmd.String("config")
			if _, err := os.Stat(configPath); err == nil {
				return fmt.Errorf("%s already exists. Delete it first or use a different path", configPath)
			}

			reader := bufio.NewReader(os.Stdin)

			// 1. Engine
			fmt.Print("Database engine (postgresql/mysql/sqlite) [postgresql]: ")
			engine, _ := reader.ReadString('\n')
			engine = strings.TrimSpace(engine)
			if engine == "" {
				engine = "postgresql"
			}

			// 2. Schema dir
			defaultSchemaDir := fmt.Sprintf("sql/migrations/%s", engine)
			fmt.Printf("Schema directory [%s]: ", defaultSchemaDir)
			schemaDir, _ := reader.ReadString('\n')
			schemaDir = strings.TrimSpace(schemaDir)
			if schemaDir == "" {
				schemaDir = defaultSchemaDir
			}

			// 3. Queries dir prefix
			defaultQueriesPrefix := fmt.Sprintf("sql/queries/%s", engine)
			fmt.Printf("Queries directory prefix [%s]: ", defaultQueriesPrefix)
			queriesPrefix, _ := reader.ReadString('\n')
			queriesPrefix = strings.TrimSpace(queriesPrefix)
			if queriesPrefix == "" {
				queriesPrefix = defaultQueriesPrefix
			}

			// 4. Output dir prefix
			defaultOutputPrefix := "internal/store/repos"
			fmt.Printf("Output directory prefix [%s]: ", defaultOutputPrefix)
			outputPrefix, _ := reader.ReadString('\n')
			outputPrefix = strings.TrimSpace(outputPrefix)
			if outputPrefix == "" {
				outputPrefix = defaultOutputPrefix
			}

			// 5. Models
			fmt.Print("Generate Go models? (y/n) [y]: ")
			modelsAnswer, _ := reader.ReadString('\n')
			modelsAnswer = strings.TrimSpace(strings.ToLower(modelsAnswer))
			if modelsAnswer == "" {
				modelsAnswer = "y"
			}

			modelsSection := ""
			if modelsAnswer == "y" || modelsAnswer == "yes" {
				modelsSection = `
    models:
      output_dir: internal/models
      output_file_name: models_gen.go
      package_name: models
      emit_json_tags: true
      emit_db_tags: true`
			}

			// 6. sqlc
			fmt.Print("Use sqlc for query code generation? (y/n) [y]: ")
			sqlcAnswer, _ := reader.ReadString('\n')
			sqlcAnswer = strings.TrimSpace(strings.ToLower(sqlcAnswer))
			if sqlcAnswer == "" {
				sqlcAnswer = "y"
			}

			sqlcSection := ""
			if sqlcAnswer == "y" || sqlcAnswer == "yes" {
				sqlcSection = `
    sqlc:
      defaults:
        sql_package: pgx/v5
        emit_interface: true
        emit_json_tags: true
        emit_db_tags: true
        emit_empty_slices: true
        emit_result_struct_pointers: true
        emit_enum_valid_method: true
        emit_all_enum_values: true`
			}

			yaml := fmt.Sprintf(`version: "2"

schemas:
  - name: main
    engine: %s
    schema_dir: %s
%s%s
    defaults:
      queries_dir_prefix: %s
      output_dir_prefix: %s
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

    # Add your tables here:
    # tables:
    #   users:
    #     primary_column: id
    #     crud:
    #       methods:
    #         create: {}
    #         get: {}
    #         find:
    #           limit: true
    #         delete: {}
`, engine, schemaDir, modelsSection, sqlcSection, queriesPrefix, outputPrefix)

			if err := os.WriteFile(configPath, []byte(yaml), 0o644); err != nil {
				return fmt.Errorf("write config: %w", err)
			}

			l.Infof("config written to %s", configPath)
			l.Info("edit the tables section and run 'pgxgen generate'")
			return nil
		},
	}
}

func newWatchCmd(l logger.Logger) *cli.Command {
	return &cli.Command{
		Name:  "watch",
		Usage: "Watch schema files and regenerate on changes",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			configPath := cmd.String("config")
			cfg, err := loadV2Config(cmd)
			if err != nil {
				return err
			}
			return watcher.Watch(ctx, l, cfg, configPath)
		},
	}
}

func newExampleCmd(_ logger.Logger) *cli.Command {
	return &cli.Command{
		Name:  "example",
		Usage: "Print an example v2 config with all features",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "engine",
				Usage: "Database engine (postgresql, mysql, sqlite)",
				Value: "postgresql",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			eng := cmd.String("engine")
			fmt.Print(exampleConfig(eng))
			return nil
		},
	}
}

func exampleConfig(engine string) string {
	return fmt.Sprintf(`# pgxgen v2 configuration
# Docs: https://github.com/tkcrm/pgxgen
version: "2"

schemas:
  - name: main
    engine: %s
    schema_dir: sql/migrations/%s

    # Go model generation
    models:
      output_dir: internal/models
      output_file_name: models_gen.go
      package_name: models
      package_path: github.com/your-org/your-project/internal/models
      emit_json_tags: true
      emit_db_tags: true
      # custom_types: [MyCustomType]  # types defined in models package

    # sqlc auto-generation (pgxgen generates sqlc.yaml automatically)
    sqlc:
      # keep_generated_config: true  # keep .pgxgen/sqlc.yaml after run (default: false)
      defaults:
        sql_package: pgx/v5
        emit_interface: true
        emit_json_tags: true
        emit_db_tags: true
        emit_empty_slices: true
        emit_result_struct_pointers: true
        emit_enum_valid_method: true
        emit_all_enum_values: true
      # overrides:
      #   rename: { d: Params }
      #   types:
      #     - db_type: uuid
      #       go_type: "github.com/google/uuid.UUID"
      #   columns:
      #     - column: users.email
      #       go_struct_tag: 'validate:"required,email"'

    # Defaults for all tables
    defaults:
      # Per-table repos: each table gets its own queries_dir and output_dir
      queries_dir_prefix: sql/queries/%s
      output_dir_prefix: internal/store/repos

      # Single repo (uncomment to use instead of per-table):
      # queries_dir: sql/queries
      # output_dir: internal/store

      crud:
        auto_clean: true
        exclude_table_name: true  # GetByID instead of GetUserByID
        methods:
          create:
            skip_columns: [id, updated_at]
            returning: "*"
            column_values: { created_at: "now()" }
      constants:
        include_column_names: true

    # Tables: unified crud + constants + sqlc per table
    tables:
      users:
        primary_column: id
        # soft_delete:
        #   column: deleted_at
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
            # batch_create:
            #   skip_columns: [id, created_at]

    # Custom SQL queries
    # custom_queries:
    #   - name: GetActiveUsers
    #     type: many
    #     table: users
    #     sql: |
    #       SELECT * FROM users
    #       WHERE is_active = true
    #       ORDER BY created_at DESC

# Custom templates (optional)
# templates:
#   crud_dir: .pgxgen/templates/crud
#   models_dir: .pgxgen/templates/models
`, engine, engine, engine)
}

func newVersionCmd(version string) *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Print the version",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Printf("pgxgen version %s (go %s)\n", version, runtime.Version())
			return nil
		},
	}
}

func newUpdateCmd(l logger.Logger, version string) *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update pgxgen to the latest version",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return updater.CheckAndUpdate(ctx, l, version)
		},
	}
}

func newFmtCmd(l logger.Logger) *cli.Command {
	return &cli.Command{
		Name:      "fmt",
		Usage:     "Format SQL files",
		ArgsUsage: "<path>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "check",
				Aliases: []string{"c"},
				Usage:   "Check formatting without modifying files (exit 1 if unformatted)",
			},
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"y"},
				Usage:   "Skip confirmation prompt",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Process files without saving (test formatting)",
			},
			&cli.StringFlag{
				Name:    "engine",
				Aliases: []string{"e"},
				Usage:   "SQL dialect (postgresql, mysql, sqlite)",
				Value:   "postgresql",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return fmt.Errorf("path argument is required\nusage: pgxgen fmt <path> [--check] [--yes] [--engine <engine>]")
			}
			target := cmd.Args().First()
			check := cmd.Bool("check")
			yes := cmd.Bool("yes")
			dryRun := cmd.Bool("dry-run")
			engine := cmd.String("engine")

			// Validate engine
			dialect := lexer.Dialect(engine)
			switch dialect {
			case lexer.DialectPostgreSQL, lexer.DialectMySQL, lexer.DialectSQLite:
			default:
				return fmt.Errorf("unsupported engine: %s (supported: postgresql, mysql, sqlite)", engine)
			}

			options := formatters.DefaultOptions()
			options.Dialect = dialect

			// Resolve target: file or directory (recursive)
			sqlFiles, err := resolveSQLFiles(target)
			if err != nil {
				return err
			}

			if len(sqlFiles) == 0 {
				l.Info("no .sql files found in", target)
				return nil
			}

			// Analyze which files need formatting
			type fileResult struct {
				path      string
				formatted []byte
			}

			var toFormat []fileResult
			var errCount int

			for _, file := range sqlFiles {
				src, err := os.ReadFile(file)
				if err != nil {
					l.Infof("error reading %s: %v", file, err)
					errCount++
					continue
				}

				formatted, err := sqlfmt.FormatFile(src, options)
				if err != nil {
					l.Infof("error formatting %s: %v", file, err)
					errCount++
					continue
				}

				if string(src) != string(formatted) {
					toFormat = append(toFormat, fileResult{path: file, formatted: formatted})
				}
			}

			// Dry-run mode: just process and report
			if dryRun {
				l.Infof("processed %d file(s), %d need formatting, %d error(s)", len(sqlFiles), len(toFormat), errCount)
				for _, f := range toFormat {
					l.Infof("  would format %s", f.path)
				}
				if errCount > 0 {
					return fmt.Errorf("%d file(s) had errors", errCount)
				}
				return nil
			}

			// Check mode: just report and exit
			if check {
				if len(toFormat) > 0 {
					l.Info("unformatted files:")
					for _, f := range toFormat {
						l.Infof("  %s", f.path)
					}
					return fmt.Errorf("%d file(s) not formatted", len(toFormat))
				}
				l.Info("all files are formatted")
				return nil
			}

			if len(toFormat) == 0 {
				l.Infof("all %d file(s) are already formatted", len(sqlFiles))
				return nil
			}

			// Show files to format
			l.Infof("files to format (%d):", len(toFormat))
			for _, f := range toFormat {
				l.Infof("  %s", f.path)
			}

			// Ask for confirmation unless --yes
			if !yes {
				fmt.Print("\nproceed? [y/N] ")
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
				if answer != "y" && answer != "yes" {
					l.Info("aborted")
					return nil
				}
			}

			// Write formatted files
			for _, f := range toFormat {
				if err := os.WriteFile(f.path, f.formatted, 0o644); err != nil {
					l.Infof("error writing %s: %v", f.path, err)
					errCount++
					continue
				}
				l.Infof("formatted %s", f.path)
			}

			if errCount > 0 {
				return fmt.Errorf("%d file(s) had errors", errCount)
			}

			return nil
		},
	}
}

// resolveSQLFiles resolves a path (file or directory) into a list of .sql files.
// For directories, it searches recursively.
func resolveSQLFiles(target string) ([]string, error) {
	info, err := os.Stat(target)
	if err != nil {
		return nil, fmt.Errorf("path %s: %w", target, err)
	}

	// Single file
	if !info.IsDir() {
		if strings.HasSuffix(strings.ToLower(info.Name()), ".sql") {
			return []string{target}, nil
		}
		return nil, fmt.Errorf("%s is not a .sql file", target)
	}

	// Directory: walk recursively
	var files []string
	err = filepath.WalkDir(target, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".sql") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk directory %s: %w", target, err)
	}

	return files, nil
}
