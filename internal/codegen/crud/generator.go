package crud

import (
	"bytes"
	"fmt"
	"slices"
	"sort"
	"strings"
	"text/template"

	"github.com/gobeam/stringy"
	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/engine"
	"github.com/tkcrm/pgxgen/internal/sqlparser/catalog"
)

// ColumnData represents a column with its placeholder.
type ColumnData struct {
	Name        string
	Placeholder string
}

// TemplateData holds all data needed to render a CRUD SQL template.
type TemplateData struct {
	MethodName      string
	OperationType   string
	TableName       string
	PrimaryColumn   string
	Columns         []ColumnData
	Returning       string
	WhereClause     string
	SoftDeleteWhere bool
	SoftDeleteColumn string
	OrderClause     string
	LimitClause     string
}

// Generator generates CRUD SQL files from config and catalog.
type Generator struct {
	eng engine.Engine
}

// New creates a new CRUD generator for the given engine.
func New(eng engine.Engine) *Generator {
	return &Generator{eng: eng}
}

// GenerateTable generates all CRUD SQL for a single table.
func (g *Generator) GenerateTable(
	tableName string,
	tableConfig config.TableConfig,
	defaultCrud *config.CrudDefaultsConfig,
	columns []string,
) ([]byte, error) {
	if tableConfig.Crud == nil || tableConfig.Crud.Methods == nil {
		return nil, nil
	}

	// Sort method keys for deterministic output
	methodKeys := make([]string, 0, len(tableConfig.Crud.Methods))
	for k := range tableConfig.Crud.Methods {
		methodKeys = append(methodKeys, k)
	}
	sort.Strings(methodKeys)

	var buf bytes.Buffer

	for _, methodName := range methodKeys {
		methodCfg := tableConfig.Crud.Methods[methodName]
		if methodCfg == nil {
			methodCfg = &config.MethodConfig{}
		}

		data, templateName, err := g.buildTemplateData(
			tableName, methodName, methodCfg, tableConfig, defaultCrud, columns,
		)
		if err != nil {
			return nil, fmt.Errorf("build template data for %s.%s: %w", tableName, methodName, err)
		}

		rendered, err := g.renderTemplate(templateName, data)
		if err != nil {
			return nil, fmt.Errorf("render template %s for %s.%s: %w", templateName, tableName, methodName, err)
		}

		buf.Write(rendered)
	}

	return buf.Bytes(), nil
}

func (g *Generator) buildTemplateData(
	tableName, methodName string,
	methodCfg *config.MethodConfig,
	tableConfig config.TableConfig,
	defaultCrud *config.CrudDefaultsConfig,
	allColumns []string,
) (*TemplateData, string, error) {
	data := &TemplateData{
		TableName:     tableName,
		PrimaryColumn: tableConfig.PrimaryColumn,
	}

	// Resolve method name
	data.MethodName = methodCfg.Name
	if data.MethodName == "" {
		data.MethodName = g.defaultMethodName(methodName, tableName, defaultCrud)
	}

	// Soft delete
	if tableConfig.SoftDelete != nil {
		data.SoftDeleteColumn = tableConfig.SoftDelete.Column
	}

	paramIndex := 1
	nextPlaceholder := func() string {
		p := g.eng.ParamPlaceholder(paramIndex)
		paramIndex++
		return p
	}

	var templateFile string

	switch methodName {
	case "create":
		templateFile = "create"
		data.OperationType = "exec"
		if methodCfg.Returning != "" {
			data.OperationType = "one"
			data.Returning = methodCfg.Returning
		}
		filtered := filterColumns(allColumns, methodCfg.SkipColumns)
		data.Columns = g.buildColumns(filtered, methodCfg.ColumnValues, &paramIndex)

	case "update":
		templateFile = "update"
		data.OperationType = "exec"
		if methodCfg.Returning != "" {
			data.OperationType = "one"
			data.Returning = methodCfg.Returning
		}
		filtered := filterColumns(allColumns, methodCfg.SkipColumns)
		data.Columns = g.buildSetColumns(filtered, methodCfg.ColumnValues, &paramIndex)
		// Add primary column to WHERE
		where := cloneWhere(methodCfg.Where)
		if tableConfig.PrimaryColumn != "" {
			if where == nil {
				where = make(map[string]config.WhereParamConfig)
			}
			where[tableConfig.PrimaryColumn] = config.WhereParamConfig{}
		}
		data.WhereClause = g.buildWhereClause(where, methodCfg.WhereAdditional, allColumns, &paramIndex)

	case "delete":
		if tableConfig.SoftDelete != nil {
			templateFile = "soft_delete"
		} else {
			templateFile = "delete"
		}
		where := cloneWhere(methodCfg.Where)
		if tableConfig.PrimaryColumn != "" {
			if where == nil {
				where = make(map[string]config.WhereParamConfig)
			}
			where[tableConfig.PrimaryColumn] = config.WhereParamConfig{}
		}
		data.WhereClause = g.buildWhereClause(where, methodCfg.WhereAdditional, allColumns, &paramIndex)

	case "get":
		templateFile = "get"
		where := cloneWhere(methodCfg.Where)
		if tableConfig.PrimaryColumn != "" {
			if where == nil {
				where = make(map[string]config.WhereParamConfig)
			}
			where[tableConfig.PrimaryColumn] = config.WhereParamConfig{}
		}
		data.WhereClause = g.buildWhereClause(where, methodCfg.WhereAdditional, allColumns, &paramIndex)
		if tableConfig.SoftDelete != nil {
			data.SoftDeleteWhere = true
		}

	case "find":
		templateFile = "find"
		data.WhereClause = g.buildWhereClause(methodCfg.Where, methodCfg.WhereAdditional, allColumns, &paramIndex)
		if tableConfig.SoftDelete != nil {
			data.SoftDeleteWhere = true
		}
		if methodCfg.Order != nil {
			data.OrderClause = fmt.Sprintf("ORDER BY %s %s", methodCfg.Order.By, methodCfg.Order.GetDirection())
		}
		if methodCfg.Limit {
			data.LimitClause = fmt.Sprintf("LIMIT %s OFFSET %s", nextPlaceholder(), nextPlaceholder())
		}

	case "total":
		templateFile = "total"
		data.WhereClause = g.buildWhereClause(methodCfg.Where, methodCfg.WhereAdditional, allColumns, &paramIndex)
		if tableConfig.SoftDelete != nil {
			data.SoftDeleteWhere = true
		}

	case "exists":
		templateFile = "exists"
		data.WhereClause = g.buildWhereClause(methodCfg.Where, methodCfg.WhereAdditional, allColumns, &paramIndex)
		if tableConfig.SoftDelete != nil {
			data.SoftDeleteWhere = true
		}

	case "batch_create":
		templateFile = "batch_create"
		filtered := filterColumns(allColumns, methodCfg.SkipColumns)
		data.Columns = g.buildColumns(filtered, methodCfg.ColumnValues, &paramIndex)

	default:
		return nil, "", fmt.Errorf("unknown method: %s", methodName)
	}

	return data, templateFile, nil
}

func (g *Generator) renderTemplate(name string, data *TemplateData) ([]byte, error) {
	tmplContent, err := getTemplate(g.eng.Name(), name)
	if err != nil {
		return nil, err
	}

	funcMap := template.FuncMap{
		"joinNames":        joinNames,
		"joinPlaceholders": joinPlaceholders,
		"joinSetClauses":   joinSetClauses,
	}

	tmpl, err := template.New(name).Funcs(funcMap).Parse(tmplContent)
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template %s: %w", name, err)
	}

	return buf.Bytes(), nil
}

func (g *Generator) defaultMethodName(method, tableName string, defaultCrud *config.CrudDefaultsConfig) string {
	excludeTableName := defaultCrud != nil && defaultCrud.ExcludeTableName

	var methodName string
	if excludeTableName {
		methodName = method
	} else {
		methodName = fmt.Sprintf("%s %s", method, tableName)
	}

	methodName = stringy.New(methodName).CamelCase().UcFirst()

	// Singularize for non-collection methods
	if !slices.Contains([]string{"find", "total"}, method) {
		if strings.HasSuffix(methodName, "s") {
			methodName = methodName[:len(methodName)-1]
		}
	}

	return methodName
}

func (g *Generator) buildColumns(columns []string, columnValues map[string]string, paramIndex *int) []ColumnData {
	result := make([]ColumnData, 0, len(columns))
	for _, col := range columns {
		cd := ColumnData{Name: col}
		if val, ok := columnValues[col]; ok {
			cd.Placeholder = val
		} else {
			cd.Placeholder = g.eng.ParamPlaceholder(*paramIndex)
			*paramIndex++
		}
		result = append(result, cd)
	}
	return result
}

func (g *Generator) buildSetColumns(columns []string, columnValues map[string]string, paramIndex *int) []ColumnData {
	return g.buildColumns(columns, columnValues, paramIndex)
}

func (g *Generator) buildWhereClause(
	where map[string]config.WhereParamConfig,
	whereAdditional []string,
	allColumns []string,
	paramIndex *int,
) string {
	if len(where) == 0 && len(whereAdditional) == 0 {
		return ""
	}

	var parts []string

	// Sort WHERE keys for deterministic output
	keys := make([]string, 0, len(where))
	for k := range where {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, col := range keys {
		wp := where[col]
		if wp.Value != "" {
			operator := wp.Operator
			if operator != "" {
				parts = append(parts, fmt.Sprintf("%s %s %s", col, operator, wp.Value))
			} else {
				parts = append(parts, fmt.Sprintf("%s %s", col, wp.Value))
			}
		} else {
			operator := wp.Operator
			if operator == "" {
				operator = "="
			}
			parts = append(parts, fmt.Sprintf("%s%s%s", col, operator, g.eng.ParamPlaceholder(*paramIndex)))
			*paramIndex++
		}
	}

	for _, additional := range whereAdditional {
		parts = append(parts, additional)
	}

	if len(parts) == 0 {
		return ""
	}

	return "WHERE " + strings.Join(parts, " AND ")
}

// Template helper functions

func joinNames(columns []ColumnData) string {
	names := make([]string, len(columns))
	for i, c := range columns {
		names[i] = c.Name
	}
	return strings.Join(names, ", ")
}

func joinPlaceholders(columns []ColumnData) string {
	placeholders := make([]string, len(columns))
	for i, c := range columns {
		placeholders[i] = c.Placeholder
	}
	return strings.Join(placeholders, ", ")
}

func joinSetClauses(columns []ColumnData) string {
	clauses := make([]string, len(columns))
	for i, c := range columns {
		clauses[i] = fmt.Sprintf("%s=%s", c.Name, c.Placeholder)
	}
	return strings.Join(clauses, ", ")
}

func filterColumns(columns, skip []string) []string {
	if len(skip) == 0 {
		return columns
	}
	result := make([]string, 0, len(columns))
	for _, col := range columns {
		if !slices.Contains(skip, col) {
			result = append(result, col)
		}
	}
	return result
}

func cloneWhere(w map[string]config.WhereParamConfig) map[string]config.WhereParamConfig {
	if w == nil {
		return nil
	}
	result := make(map[string]config.WhereParamConfig, len(w))
	for k, v := range w {
		result[k] = v
	}
	return result
}

// GetTableColumns extracts column names from a catalog table.
func GetTableColumns(cat *catalog.Catalog, tableName string) []string {
	for _, schema := range cat.Schemas {
		for _, table := range schema.Tables {
			if table.Name == tableName {
				columns := make([]string, len(table.Columns))
				for i, col := range table.Columns {
					columns[i] = col.Name
				}
				return columns
			}
		}
	}
	return nil
}
