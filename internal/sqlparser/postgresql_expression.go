package sqlparser

import (
	"strings"

	pg "github.com/pganalyze/pg_query_go/v6"
	"github.com/tkcrm/pgxgen/internal/sqlparser/catalog"
)

// resolvePostgresExpression infers the catalog type of an expression used as a
// view column. pg_query parses syntax but does not perform PostgreSQL's semantic
// type analysis, so built-in functions and polymorphic aggregates are resolved
// from their documented return-type rules here.
func (p *postgresParser) resolvePostgresExpression(
	cat *catalog.Catalog,
	node *pg.Node,
	fromTables map[string]string,
) *catalog.Column {
	if node == nil {
		return unknownPostgresExpression()
	}

	switch n := node.Node.(type) {
	case *pg.Node_ColumnRef:
		tableName, columnName := p.extractColumnRef(n.ColumnRef)
		if col := p.findColumn(cat, tableName, columnName, fromTables); col != nil {
			return clonePostgresExpression(col)
		}
	case *pg.Node_TypeCast:
		if n.TypeCast.TypeName == nil {
			break
		}
		arg := p.resolvePostgresExpression(cat, n.TypeCast.Arg, fromTables)
		return &catalog.Column{
			Type:      pgTypeName(n.TypeCast.TypeName),
			FullType:  deparseTypeName(n.TypeCast.TypeName),
			NotNull:   arg.NotNull,
			IsArray:   len(n.TypeCast.TypeName.ArrayBounds) > 0,
			ArrayDims: len(n.TypeCast.TypeName.ArrayBounds),
		}
	case *pg.Node_FuncCall:
		if col := p.resolvePostgresBuiltinFunction(cat, n.FuncCall, fromTables); col != nil {
			return col
		}
	case *pg.Node_SubLink:
		return p.resolvePostgresScalarSubquery(cat, n.SubLink)
	case *pg.Node_CoalesceExpr:
		return p.resolvePostgresCommonType(cat, n.CoalesceExpr.Args, fromTables, true)
	case *pg.Node_MinMaxExpr:
		return p.resolvePostgresCommonType(cat, n.MinMaxExpr.Args, fromTables, true)
	case *pg.Node_SqlvalueFunction:
		return resolvePostgresSQLValueFunction(n.SqlvalueFunction.Op)
	case *pg.Node_AConst:
		return resolvePostgresConstant(n.AConst)
	case *pg.Node_NamedArgExpr:
		return p.resolvePostgresExpression(cat, n.NamedArgExpr.Arg, fromTables)
	case *pg.Node_CollateClause:
		return p.resolvePostgresExpression(cat, n.CollateClause.Arg, fromTables)
	case *pg.Node_NullTest, *pg.Node_BooleanTest:
		return postgresExpression("boolean", true)
	}

	return unknownPostgresExpression()
}

func (p *postgresParser) resolvePostgresBuiltinFunction(
	cat *catalog.Catalog,
	call *pg.FuncCall,
	fromTables map[string]string,
) *catalog.Column {
	name, builtin := postgresBuiltinFunctionName(call.Funcname)
	if !builtin {
		return nil
	}

	args := make([]*catalog.Column, 0, len(call.Args))
	for _, arg := range call.Args {
		args = append(args, p.resolvePostgresExpression(cat, arg, fromTables))
	}

	switch name {
	case "count":
		return postgresExpression("bigint", true)
	case "sum":
		return postgresExpression(postgresSumType(postgresFirstType(args)), false)
	case "avg":
		return postgresExpression(postgresAverageType(postgresFirstType(args)), false)
	case "min", "max":
		return postgresArgumentType(args, 0, false)
	case "array_agg":
		col := postgresArgumentType(args, 0, false)
		if col.Type == "any" {
			return col
		}
		col.IsArray = true
		col.ArrayDims++
		return col
	case "string_agg":
		if postgresFirstType(args) == "bytea" {
			return postgresExpression("bytea", false)
		}
		return postgresExpression("text", false)
	case "bool_and", "bool_or", "every":
		return postgresExpression("boolean", false)
	case "json_agg", "json_object_agg":
		return postgresExpression("json", false)
	case "jsonb_agg", "jsonb_object_agg":
		return postgresExpression("jsonb", false)
	case "bit_and", "bit_or", "bit_xor":
		return postgresArgumentType(args, 0, false)
	case "corr", "covar_pop", "covar_samp", "regr_avgx", "regr_avgy", "regr_intercept",
		"regr_r2", "regr_slope", "regr_sxx", "regr_sxy", "regr_syy":
		return postgresExpression("double precision", false)
	case "regr_count":
		return postgresExpression("bigint", true)
	case "stddev", "stddev_pop", "stddev_samp", "variance", "var_pop", "var_samp":
		return postgresExpression(postgresStatisticsType(postgresFirstType(args)), false)
	case "row_number", "rank", "dense_rank":
		return postgresExpression("bigint", true)
	case "percent_rank", "cume_dist":
		return postgresExpression("double precision", true)
	case "ntile":
		return postgresExpression("integer", true)
	case "lag", "lead", "first_value", "last_value", "nth_value":
		return postgresArgumentType(args, 0, false)
	case "lower", "upper", "initcap", "btrim", "ltrim", "rtrim", "reverse", "md5",
		"quote_ident", "quote_literal", "quote_nullable", "to_char":
		return postgresStrictExpression("text", args)
	case "concat":
		return postgresExpression("text", true)
	case "format":
		return postgresStrictExpression("text", args)
	case "concat_ws":
		return postgresExpression("text", len(args) > 0 && args[0].NotNull)
	case "length", "char_length", "character_length", "octet_length", "bit_length",
		"array_length", "array_ndims", "cardinality", "json_array_length", "jsonb_array_length":
		return postgresStrictExpression("integer", args)
	case "date_part":
		return postgresStrictExpression("double precision", args)
	case "date_trunc":
		if len(args) < 2 {
			return unknownPostgresExpression()
		}
		typ := normalizePostgresExpressionType(args[1].Type)
		if typ != "timestamp" && typ != "timestamptz" && typ != "interval" {
			return unknownPostgresExpression()
		}
		return postgresStrictExpression(typ, args)
	case "to_date":
		return postgresStrictExpression("date", args)
	case "to_timestamp":
		return postgresStrictExpression("timestamptz", args)
	case "now", "transaction_timestamp", "statement_timestamp", "clock_timestamp":
		return postgresExpression("timestamptz", true)
	case "timeofday":
		return postgresExpression("text", true)
	case "gen_random_uuid":
		return postgresExpression("uuid", true)
	case "random":
		return postgresExpression("double precision", true)
	case "pg_backend_pid":
		return postgresExpression("integer", true)
	case "current_schema":
		return postgresExpression("text", false)
	case "current_schemas":
		return &catalog.Column{Type: "text", FullType: "text[]", NotNull: true, IsArray: true, ArrayDims: 1}
	case "json_build_array", "json_build_object":
		return postgresExpression("json", true)
	case "row_to_json", "to_json":
		return postgresStrictExpression("json", args)
	case "jsonb_build_array", "jsonb_build_object":
		return postgresExpression("jsonb", true)
	case "to_jsonb":
		return postgresStrictExpression("jsonb", args)
	case "abs", "ceil", "ceiling", "floor", "round", "trunc":
		return postgresArgumentType(args, 0, postgresAllNotNull(args))
	case "array_append", "array_prepend", "array_cat":
		for _, arg := range args {
			if arg.IsArray {
				col := clonePostgresExpression(arg)
				col.NotNull = postgresAllNotNull(args)
				return col
			}
		}
	}

	return nil
}

func (p *postgresParser) resolvePostgresScalarSubquery(cat *catalog.Catalog, link *pg.SubLink) *catalog.Column {
	if link == nil || link.Subselect == nil {
		return unknownPostgresExpression()
	}
	selNode, ok := link.Subselect.Node.(*pg.Node_SelectStmt)
	if !ok || len(selNode.SelectStmt.TargetList) != 1 {
		return unknownPostgresExpression()
	}
	target, ok := selNode.SelectStmt.TargetList[0].Node.(*pg.Node_ResTarget)
	if !ok {
		return unknownPostgresExpression()
	}

	sel := selNode.SelectStmt
	col := p.resolvePostgresExpression(cat, target.ResTarget.Val, p.extractFromTables(sel))
	if !postgresScalarSubqueryGuaranteedRow(sel, target.ResTarget.Val) {
		col.NotNull = false
	}
	return col
}

func postgresScalarSubqueryGuaranteedRow(sel *pg.SelectStmt, expression *pg.Node) bool {
	if sel == nil || sel.Op != pg.SetOperation_SETOP_NONE || len(sel.GroupClause) > 0 || sel.HavingClause != nil ||
		sel.LimitCount != nil || sel.LimitOffset != nil {
		return false
	}
	if len(sel.FromClause) == 0 && sel.WhereClause == nil {
		return true
	}
	callNode, ok := expression.Node.(*pg.Node_FuncCall)
	if !ok || callNode.FuncCall.Over != nil {
		return false
	}
	name, builtin := postgresBuiltinFunctionName(callNode.FuncCall.Funcname)
	return builtin && postgresAggregateFunction(name)
}

func postgresAggregateFunction(name string) bool {
	switch name {
	case "count", "sum", "avg", "min", "max", "array_agg", "string_agg", "bool_and", "bool_or", "every",
		"json_agg", "json_object_agg", "jsonb_agg", "jsonb_object_agg", "bit_and", "bit_or", "bit_xor",
		"corr", "covar_pop", "covar_samp", "regr_avgx", "regr_avgy", "regr_count", "regr_intercept",
		"regr_r2", "regr_slope", "regr_sxx", "regr_sxy", "regr_syy", "stddev", "stddev_pop",
		"stddev_samp", "variance", "var_pop", "var_samp":
		return true
	default:
		return false
	}
}

func (p *postgresParser) resolvePostgresCommonType(
	cat *catalog.Catalog,
	nodes []*pg.Node,
	fromTables map[string]string,
	notNullIfAny bool,
) *catalog.Column {
	result := unknownPostgresExpression()
	for _, node := range nodes {
		arg := p.resolvePostgresExpression(cat, node, fromTables)
		if result.Type == "any" && arg.Type != "any" {
			result = clonePostgresExpression(arg)
		}
		if notNullIfAny && arg.NotNull {
			result.NotNull = true
		}
	}
	return result
}

func resolvePostgresSQLValueFunction(op pg.SQLValueFunctionOp) *catalog.Column {
	switch op {
	case pg.SQLValueFunctionOp_SVFOP_CURRENT_DATE:
		return postgresExpression("date", true)
	case pg.SQLValueFunctionOp_SVFOP_CURRENT_TIME, pg.SQLValueFunctionOp_SVFOP_CURRENT_TIME_N:
		return postgresExpression("pg_catalog.timetz", true)
	case pg.SQLValueFunctionOp_SVFOP_CURRENT_TIMESTAMP, pg.SQLValueFunctionOp_SVFOP_CURRENT_TIMESTAMP_N:
		return postgresExpression("timestamptz", true)
	case pg.SQLValueFunctionOp_SVFOP_LOCALTIME, pg.SQLValueFunctionOp_SVFOP_LOCALTIME_N:
		return postgresExpression("pg_catalog.time", true)
	case pg.SQLValueFunctionOp_SVFOP_LOCALTIMESTAMP, pg.SQLValueFunctionOp_SVFOP_LOCALTIMESTAMP_N:
		return postgresExpression("timestamp", true)
	case pg.SQLValueFunctionOp_SVFOP_CURRENT_ROLE, pg.SQLValueFunctionOp_SVFOP_CURRENT_USER,
		pg.SQLValueFunctionOp_SVFOP_USER, pg.SQLValueFunctionOp_SVFOP_SESSION_USER,
		pg.SQLValueFunctionOp_SVFOP_CURRENT_CATALOG, pg.SQLValueFunctionOp_SVFOP_CURRENT_SCHEMA:
		return postgresExpression("text", true)
	default:
		return unknownPostgresExpression()
	}
}

func resolvePostgresConstant(value *pg.A_Const) *catalog.Column {
	if value == nil || value.Isnull {
		return unknownPostgresExpression()
	}
	switch value.Val.(type) {
	case *pg.A_Const_Ival:
		return postgresExpression("integer", true)
	case *pg.A_Const_Fval:
		return postgresExpression("numeric", true)
	case *pg.A_Const_Boolval:
		return postgresExpression("boolean", true)
	case *pg.A_Const_Sval:
		return postgresExpression("text", true)
	case *pg.A_Const_Bsval:
		return postgresExpression("bit", true)
	default:
		return unknownPostgresExpression()
	}
}

func postgresBuiltinFunctionName(parts []*pg.Node) (string, bool) {
	if len(parts) == 0 {
		return "", false
	}
	if len(parts) > 1 && !strings.EqualFold(pgStringVal(parts[len(parts)-2]), "pg_catalog") {
		return "", false
	}
	return strings.ToLower(pgStringVal(parts[len(parts)-1])), true
}

func postgresSumType(argType string) string {
	switch normalizePostgresExpressionType(argType) {
	case "smallint", "integer":
		return "bigint"
	case "bigint", "numeric":
		return "numeric"
	case "real":
		return "real"
	case "double precision":
		return "double precision"
	case "money":
		return "money"
	case "interval":
		return "interval"
	default:
		return "any"
	}
}

func postgresAverageType(argType string) string {
	switch normalizePostgresExpressionType(argType) {
	case "smallint", "integer", "bigint", "numeric":
		return "numeric"
	case "real", "double precision":
		return "double precision"
	case "interval":
		return "interval"
	default:
		return "any"
	}
}

func postgresStatisticsType(argType string) string {
	switch normalizePostgresExpressionType(argType) {
	case "smallint", "integer", "bigint", "numeric":
		return "numeric"
	case "real", "double precision":
		return "double precision"
	default:
		return "any"
	}
}

func normalizePostgresExpressionType(typ string) string {
	switch strings.ToLower(strings.TrimPrefix(typ, "pg_catalog.")) {
	case "int2":
		return "smallint"
	case "int", "int4":
		return "integer"
	case "int8":
		return "bigint"
	case "decimal":
		return "numeric"
	case "float4":
		return "real"
	case "float", "float8":
		return "double precision"
	case "timestamp without time zone":
		return "timestamp"
	case "timestamp with time zone":
		return "timestamptz"
	default:
		return strings.ToLower(strings.TrimPrefix(typ, "pg_catalog."))
	}
}

func postgresArgumentType(args []*catalog.Column, index int, notNull bool) *catalog.Column {
	if index < 0 || index >= len(args) {
		return unknownPostgresExpression()
	}
	col := clonePostgresExpression(args[index])
	col.NotNull = notNull
	return col
}

func postgresStrictExpression(typ string, args []*catalog.Column) *catalog.Column {
	return postgresExpression(typ, postgresAllNotNull(args))
}

func postgresAllNotNull(args []*catalog.Column) bool {
	if len(args) == 0 {
		return true
	}
	for _, arg := range args {
		if !arg.NotNull {
			return false
		}
	}
	return true
}

func postgresFirstType(args []*catalog.Column) string {
	if len(args) == 0 {
		return "any"
	}
	return args[0].Type
}

func postgresExpression(typ string, notNull bool) *catalog.Column {
	if typ == "" {
		typ = "any"
	}
	return &catalog.Column{Type: typ, FullType: typ, NotNull: notNull}
}

func unknownPostgresExpression() *catalog.Column {
	return postgresExpression("any", false)
}

func clonePostgresExpression(col *catalog.Column) *catalog.Column {
	if col == nil {
		return unknownPostgresExpression()
	}
	return &catalog.Column{
		Type:      col.Type,
		FullType:  col.FullType,
		NotNull:   col.NotNull,
		IsArray:   col.IsArray,
		ArrayDims: col.ArrayDims,
		Length:    col.Length,
	}
}
