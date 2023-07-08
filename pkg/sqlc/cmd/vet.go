package cmd

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/trace"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/ext"
	"github.com/jackc/pgx/v5"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"

	"github.com/tkcrm/pgxgen/pkg/sqlc/config"
	"github.com/tkcrm/pgxgen/pkg/sqlc/debug"
	"github.com/tkcrm/pgxgen/pkg/sqlc/opts"
	"github.com/tkcrm/pgxgen/pkg/sqlc/plugin"
	"github.com/tkcrm/pgxgen/pkg/sqlc/shfmt"
	"github.com/tkcrm/pgxgen/pkg/sqlc/sql/ast"
)

var ErrFailedChecks = errors.New("failed checks")

const RuleDbPrepare = "sqlc/db-prepare"

func NewCmdVet() *cobra.Command {
	return &cobra.Command{
		Use:   "vet",
		Short: "Vet examines queries",
		RunE: func(cmd *cobra.Command, args []string) error {
			defer trace.StartRegion(cmd.Context(), "vet").End()
			stderr := cmd.ErrOrStderr()
			dir, name := getConfigPath(stderr, cmd.Flag("file"))
			if err := Vet(cmd.Context(), ParseEnv(cmd), dir, name, stderr); err != nil {
				if !errors.Is(err, ErrFailedChecks) {
					fmt.Fprintf(stderr, "%s\n", err)
				}
				os.Exit(1)
			}
			return nil
		},
	}
}

type emptyProgram struct {
}

func (e *emptyProgram) Eval(any) (ref.Val, *cel.EvalDetails, error) {
	return nil, nil, fmt.Errorf("unimplemented")
}

func (e *emptyProgram) ContextEval(ctx context.Context, a any) (ref.Val, *cel.EvalDetails, error) {
	return e.Eval(a)
}

func Vet(ctx context.Context, e Env, dir, filename string, stderr io.Writer) error {
	configPath, conf, err := readConfig(stderr, dir, filename)
	if err != nil {
		return err
	}

	base := filepath.Base(configPath)
	if err := config.Validate(conf); err != nil {
		fmt.Fprintf(stderr, "error validating %s: %s\n", base, err)
		return err
	}

	if err := e.Validate(conf); err != nil {
		fmt.Fprintf(stderr, "error validating %s: %s\n", base, err)
		return err
	}

	env, err := cel.NewEnv(
		cel.StdLib(),
		ext.Strings(ext.StringsVersion(1)),
		cel.Types(
			&plugin.VetConfig{},
			&plugin.VetQuery{},
		),
		cel.Variable("query",
			cel.ObjectType("plugin.VetQuery"),
		),
		cel.Variable("config",
			cel.ObjectType("plugin.VetConfig"),
		),
	)
	if err != nil {
		return fmt.Errorf("new env: %s", err)
	}

	checks := map[string]cel.Program{
		RuleDbPrepare: &emptyProgram{},
	}
	msgs := map[string]string{}

	for _, c := range conf.Rules {
		if c.Name == "" {
			return fmt.Errorf("checks require a name")
		}
		if _, found := checks[c.Name]; found {
			return fmt.Errorf("type-check error: a check with the name '%s' already exists", c.Name)
		}
		if c.Rule == "" {
			return fmt.Errorf("type-check error: %s is empty", c.Name)
		}
		ast, issues := env.Compile(c.Rule)
		if issues != nil && issues.Err() != nil {
			return fmt.Errorf("type-check error: %s %s", c.Name, issues.Err())
		}
		prg, err := env.Program(ast)
		if err != nil {
			return fmt.Errorf("program construction error: %s %s", c.Name, err)
		}
		checks[c.Name] = prg
		msgs[c.Name] = c.Msg
	}

	c := checker{
		Checks:     checks,
		Conf:       conf,
		Dir:        dir,
		Env:        env,
		Envmap:     map[string]string{},
		Msgs:       msgs,
		Stderr:     stderr,
		NoDatabase: e.NoDatabase,
	}
	errored := false
	for _, sql := range conf.SQL {
		if err := c.checkSQL(ctx, sql); err != nil {
			if !errors.Is(err, ErrFailedChecks) {
				fmt.Fprintf(stderr, "%s\n", err)
			}
			errored = true
		}
	}
	if errored {
		return ErrFailedChecks
	}
	return nil
}

// Determine if a query can be prepared based on the engine and the statement
// type.
func prepareable(sql config.SQL, raw *ast.RawStmt) bool {
	if sql.Engine == config.EnginePostgreSQL {
		// TOOD: Add support for MERGE and VALUES stmts
		switch raw.Stmt.(type) {
		case *ast.DeleteStmt:
			return true
		case *ast.InsertStmt:
			return true
		case *ast.SelectStmt:
			return true
		case *ast.UpdateStmt:
			return true
		default:
			return false
		}
	}
	// Almost all statements in MySQL can be prepared, so I'm just going to assume they can be
	// https://dev.mysql.com/doc/refman/8.0/en/sql-prepared-statements.html
	if sql.Engine == config.EngineMySQL {
		return true
	}
	if sql.Engine == config.EngineSQLite {
		return true
	}
	return false
}

type preparer interface {
	Prepare(context.Context, string, string) error
}

type pgxPreparer struct {
	c *pgx.Conn
}

func (p *pgxPreparer) Prepare(ctx context.Context, name, query string) error {
	_, err := p.c.Prepare(ctx, name, query)
	return err
}

type dbPreparer struct {
	db *sql.DB
}

func (p *dbPreparer) Prepare(ctx context.Context, name, query string) error {
	_, err := p.db.PrepareContext(ctx, query)
	return err
}

type checker struct {
	Checks     map[string]cel.Program
	Conf       *config.Config
	Dir        string
	Env        *cel.Env
	Envmap     map[string]string
	Msgs       map[string]string
	Stderr     io.Writer
	NoDatabase bool
}

func (c *checker) DSN(dsn string) (string, error) {
	// Populate the environment variable map if it is empty
	if len(c.Envmap) == 0 {
		for _, e := range os.Environ() {
			k, v, _ := strings.Cut(e, "=")
			c.Envmap[k] = v
		}
	}
	return shfmt.Replace(dsn, c.Envmap), nil
}

func (c *checker) checkSQL(ctx context.Context, s config.SQL) error {
	// TODO: Create a separate function for this logic so we can
	combo := config.Combine(*c.Conf, s)

	// TODO: This feels like a hack that will bite us later
	joined := make([]string, 0, len(s.Schema))
	for _, s := range s.Schema {
		joined = append(joined, filepath.Join(c.Dir, s))
	}
	s.Schema = joined

	joined = make([]string, 0, len(s.Queries))
	for _, q := range s.Queries {
		joined = append(joined, filepath.Join(c.Dir, q))
	}
	s.Queries = joined

	var name string
	parseOpts := opts.Parser{
		Debug: debug.Debug,
	}

	result, failed := parse(ctx, name, c.Dir, s, combo, parseOpts, c.Stderr)
	if failed {
		return ErrFailedChecks
	}

	// TODO: Add MySQL support
	var prep preparer
	if s.Database != nil {
		if c.NoDatabase {
			return fmt.Errorf("database: connections disabled via command line flag")
		}
		dburl, err := c.DSN(s.Database.URI)
		if err != nil {
			return err
		}
		switch s.Engine {
		case config.EnginePostgreSQL:
			conn, err := pgx.Connect(ctx, dburl)
			if err != nil {
				return fmt.Errorf("database: connection error: %s", err)
			}
			if err := conn.Ping(ctx); err != nil {
				return fmt.Errorf("database: connection error: %s", err)
			}
			defer conn.Close(ctx)
			prep = &pgxPreparer{conn}
		case config.EngineMySQL:
			db, err := sql.Open("mysql", dburl)
			if err != nil {
				return fmt.Errorf("database: connection error: %s", err)
			}
			if err := db.PingContext(ctx); err != nil {
				return fmt.Errorf("database: connection error: %s", err)
			}
			defer db.Close()
			prep = &dbPreparer{db}
		case config.EngineSQLite:
			db, err := sql.Open("sqlite3", dburl)
			if err != nil {
				return fmt.Errorf("database: connection error: %s", err)
			}
			if err := db.PingContext(ctx); err != nil {
				return fmt.Errorf("database: connection error: %s", err)
			}
			defer db.Close()
			prep = &dbPreparer{db}
		default:
			return fmt.Errorf("unsupported database uri: %s", s.Engine)
		}
	}

	errored := false
	req := codeGenRequest(result, combo)
	cfg := vetConfig(req)
	for i, query := range req.Queries {
		q := vetQuery(query)
		for _, name := range s.Rules {
			// Built-in rule
			if name == RuleDbPrepare {
				if prep == nil {
					fmt.Fprintf(c.Stderr, "%s: %s: %s: error preparing query: database connection required\n", query.Filename, q.Name, name)
					errored = true
					continue
				}
				original := result.Queries[i]
				if prepareable(s, original.RawStmt) {
					name := fmt.Sprintf("sqlc_vet_%d_%d", time.Now().Unix(), i)
					if err := prep.Prepare(ctx, name, query.Text); err != nil {
						fmt.Fprintf(c.Stderr, "%s: %s: %s: error preparing query: %s\n", query.Filename, q.Name, name, err)
						errored = true
					}
				}
				continue
			}

			prg, ok := c.Checks[name]
			if !ok {
				return fmt.Errorf("type-check error: a check with the name '%s' does not exist", name)
			}
			out, _, err := prg.Eval(map[string]any{
				"query":  q,
				"config": cfg,
			})
			if err != nil {
				return err
			}
			tripped, ok := out.Value().(bool)
			if !ok {
				return fmt.Errorf("expression returned non-bool value: %v", out.Value())
			}
			if tripped {
				// TODO: Get line numbers in the output
				msg := c.Msgs[name]
				if msg == "" {
					fmt.Fprintf(c.Stderr, "%s: %s: %s\n", query.Filename, q.Name, name)
				} else {
					fmt.Fprintf(c.Stderr, "%s: %s: %s: %s\n", query.Filename, q.Name, name, msg)
				}
				errored = true
			}
		}
	}
	if errored {
		return ErrFailedChecks
	}
	return nil
}

func vetConfig(req *plugin.CodeGenRequest) *plugin.VetConfig {
	return &plugin.VetConfig{
		Version: req.Settings.Version,
		Engine:  req.Settings.Engine,
		Schema:  req.Settings.Schema,
		Queries: req.Settings.Queries,
	}
}

func vetQuery(q *plugin.Query) *plugin.VetQuery {
	var params []*plugin.VetParameter
	for _, p := range q.Params {
		params = append(params, &plugin.VetParameter{
			Number: p.Number,
		})
	}
	return &plugin.VetQuery{
		Sql:    q.Text,
		Name:   q.Name,
		Cmd:    strings.TrimPrefix(":", q.Cmd),
		Params: params,
	}
}
