package gen

import (
	"fmt"
	"io"
	"text/template"

	"github.com/tkcrm/pgxgen/gen/internal/config"
	"github.com/tkcrm/pgxgen/gen/internal/meta"
)

// genInterfaces emits the DBQueries interface shared between the generated PGClient
// and the generated TxPGClient. This allows user code to be written in such a way to
func (g *Generator) genInterfaces(into io.Writer, conf *config.DbConfig) error {
	g.log.Infof("	generating DBQueries interface\n")

	var genCtx ifaceGenCtx

	// populate tables
	genCtx.Tables = make([]tableIfaceGenCtx, 0, len(conf.Tables))
	for _, tc := range conf.Tables {
		tableInfo, ok := g.metaResolver.TableMeta(tc.Name)
		if !ok {
			return fmt.Errorf("could get schema info about table '%s'", tc.Name)
		}

		genCtx.Tables = append(genCtx.Tables, tableIfaceGenCtx{
			GoName:   tableInfo.Info.GoName,
			PkeyType: tableInfo.Info.PkeyCol.TypeInfo.Name,
		})
	}

	// poplulate queries
	genCtx.Queries = make([]meta.QueryMeta, 0, len(conf.Queries))
	for i := range conf.Queries {
		meta, err := g.metaResolver.QueryMeta(&conf.Queries[i], true /* inferArgTypes */)
		if err != nil {
			return err
		}
		genCtx.Queries = append(genCtx.Queries, meta)
	}

	// populate the statement gen ctx
	genCtx.Stmts = make([]meta.StmtMeta, 0, len(conf.Stmts))
	for i := range conf.Stmts {
		meta, err := g.metaResolver.StmtMeta(&conf.Stmts[i])
		if err != nil {
			return err
		}
		genCtx.Stmts = append(genCtx.Stmts, meta)
	}

	return dbQueriesTmpl.Execute(into, genCtx)
}

type tableIfaceGenCtx struct {
	GoName   string
	PkeyType string
}

type ifaceGenCtx struct {
	Tables      []tableIfaceGenCtx
	Queries     []meta.QueryMeta
	StoredFuncs []meta.QueryMeta
	Stmts       []meta.StmtMeta
}

var dbQueriesTmpl *template.Template = template.Must(template.New("db-queries-tmpl").Parse(`

type DBQueries interface {
	//
	// automatic CRUD methods
	//

	{{ range .Tables }}
	// {{ .GoName }} methods
	Get{{ .GoName }}(ctx context.Context, id {{ .PkeyType }}, opts ...pgxgen.GetOpt) (*{{ .GoName }}, error)
	List{{ .GoName }}(ctx context.Context, ids []{{ .PkeyType }}, opts ...pgxgen.ListOpt) ([]{{ .GoName }}, error)
	Insert{{ .GoName }}(ctx context.Context, value *{{ .GoName }}, opts ...pgxgen.InsertOpt) ({{ .PkeyType }}, error)
	BulkInsert{{ .GoName }}(ctx context.Context, values []{{ .GoName }}, opts ...pgxgen.InsertOpt) ([]{{ .PkeyType }}, error)
	Update{{ .GoName }}(ctx context.Context, value *{{ .GoName }}, fieldMask pgxgen.FieldSet, opts ...pgxgen.UpdateOpt) (ret {{ .PkeyType }}, err error)
	Upsert{{ .GoName }}(ctx context.Context, value *{{ .GoName }}, constraintNames []string, fieldMask pgxgen.FieldSet, opts ...pgxgen.UpsertOpt) ({{ .PkeyType }}, error)
	BulkUpsert{{ .GoName }}(ctx context.Context, values []{{ .GoName }}, constraintNames []string, fieldMask pgxgen.FieldSet, opts ...pgxgen.UpsertOpt) ([]{{ .PkeyType }}, error)
	Delete{{ .GoName }}(ctx context.Context, id {{ .PkeyType }}, opts ...pgxgen.DeleteOpt) error
	BulkDelete{{ .GoName }}(ctx context.Context, ids []{{ .PkeyType }}, opts ...pgxgen.DeleteOpt) error
	{{ .GoName }}FillIncludes(ctx context.Context, rec *{{ .GoName }}, includes *include.Spec, opts ...pgxgen.IncludeOpt) error
	{{ .GoName }}BulkFillIncludes(ctx context.Context, recs []*{{ .GoName }}, includes *include.Spec, opts ...pgxgen.IncludeOpt) error
	{{ end }}

	//
	// query methods
	//

	{{ range $i, $query := .Queries }}
	{{ if .ConfigData.SingleResult }}
	// {{ .ConfigData.Name }} query
	{{ .ConfigData.Name }}(
		ctx context.Context,
		{{- range .Args }}
		{{- if $query.ConfigData.NullableArguments }}
		{{ .GoName }} {{ .TypeInfo.NullName }},
		{{- else }}
		{{ .GoName }} {{ .TypeInfo.Name }},
		{{- end }}
		{{- end }}
	{{- if (not .MultiReturn) }}
	) ({{ .ReturnTypeName }}, error)
	{{- else }}
	) (*{{ .ReturnTypeName }}, error)
	{{- end }}

	{{ else }}
	// {{ .ConfigData.Name }} query
	{{ .ConfigData.Name }}(
		ctx context.Context,
		{{- range .Args }}
		{{- if $query.ConfigData.NullableArguments }}
		{{ .GoName }} {{ .TypeInfo.NullName }},
		{{- else }}
		{{ .GoName }} {{ .TypeInfo.Name }},
		{{- end }}
		{{- end }}
	) ([]{{ .ReturnTypeName }}, error)
	{{ .ConfigData.Name }}Query(
		ctx context.Context,
		{{- range .Args }}
		{{ .GoName }} {{ .TypeInfo.Name }},
		{{- end }}
	) (*sql.Rows, error)
	{{ end }}
	{{ end }}

	//
	// stored function methods
	//

	{{ range .StoredFuncs }}
	// {{ .ConfigData.Name }} stored function
	{{ .ConfigData.Name }}(
		ctx context.Context,
		{{- range .Args }}
		{{ .GoName }} {{ .TypeInfo.Name }},
		{{- end }}
	) ([]{{ .ReturnTypeName }}, error)
	{{ .ConfigData.Name }}Query(
		ctx context.Context,
		{{- range .Args }}
		{{ .GoName }} {{ .TypeInfo.Name }},
		{{- end }}
	) (*sql.Rows, error)
	{{ end }}

	//
	// stmt methods
	//

	{{ range .Stmts }}
	// {{ .ConfigData.Name }} stmt
	{{ .ConfigData.Name }}(
		ctx context.Context,
		{{- range .Args}}
		{{ .GoName }} {{ .TypeInfo.Name }},
		{{- end}}
	) (sql.Result, error)
	{{ end }}
}

`))
