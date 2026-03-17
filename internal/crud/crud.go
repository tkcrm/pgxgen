package crud

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/gobeam/stringy"
	cmnutils "github.com/tkcrm/modules/pkg/utils"
	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/generator"
	"github.com/tkcrm/pgxgen/internal/schema"
	"github.com/tkcrm/pgxgen/pkg/logger"
	"github.com/tkcrm/pgxgen/utils"
)

type crud struct {
	logger   logger.Logger
	config   config.Config
	catalogs map[string]schema.CatalogResult

	pgxgenFileDir string
}

func New(logger logger.Logger, cfg config.Config) generator.IGenerator {
	return &crud{
		logger: logger,
		config: cfg,
	}
}

func (s *crud) Generate(_ context.Context, args []string) error {
	for _, cfg := range s.config.Pgxgen.Sqlc {
		s.logger.Infof("generate crud for schema: %s", cfg.SchemaDir)
		timeStart := time.Now()

		// validate config
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("config validation error: %w", err)
		}

		pgxgenAbsFilePath, err := filepath.Abs(s.config.ConfigPaths.PgxgenConfigFilePath)
		if err != nil {
			return fmt.Errorf("failed to get sqlc config abs file path: %w", err)
		}

		s.pgxgenFileDir = filepath.Dir(pgxgenAbsFilePath)

		sqlcConfigDir := filepath.Dir(s.config.ConfigPaths.SqlcConfigFilePath)

		// get queries paths for current schema
		queriesPaths, err := config.GetPathsByScheme(s.config.Sqlc.GetPaths(), cfg.SchemaDir, "queries", sqlcConfigDir)
		if err != nil {
			return fmt.Errorf("GetPathsByScheme error: %w", err)
		}

		// get output paths for current schema
		outputPaths, err := config.GetPathsByScheme(s.config.Sqlc.GetPaths(), cfg.SchemaDir, "out", sqlcConfigDir)
		if err != nil {
			return fmt.Errorf("GetPathsByScheme error: %w", err)
		}

		engines, err := config.GetEnginesByScheme(s.config.Sqlc.GetPaths(), cfg.SchemaDir, sqlcConfigDir)
		if err != nil {
			return fmt.Errorf("GetEnginesByScheme error: %w", err)
		}

		if len(engines) != len(outputPaths) {
			return fmt.Errorf("engines count does not match output paths count")
		}

		// get catalogs
		allCatalogs, err := schema.GetCatalogs(s.config.Sqlc, sqlcConfigDir)
		if err != nil {
			return fmt.Errorf("getCatalogs error: %w", err)
		}

		s.catalogs = make(map[string]schema.CatalogResult, len(outputPaths))
		for _, path := range outputPaths {
			item, err := schema.GetCatalogByOutputDir(allCatalogs, path)
			if err != nil {
				return err
			}
			s.catalogs[path] = item
		}

		params := make([]generateSQLForEachTableParams, len(outputPaths))
		for i, p := range outputPaths {
			params[i] = generateSQLForEachTableParams{
				outputPath: p,
				engine:     engines[i],
			}
		}

		// get sql code for each tables
		sqlData, err := s.generateSQLForEachTable(cfg.CrudParams, params)
		if err != nil {
			return fmt.Errorf("generate sql for each tables error: %w", err)
		}

		// remove generated files
		if cfg.CrudParams.AutoRemoveGeneratedFiles {
			for _, p := range queriesPaths {
				if err := utils.RemoveFiles(p, "_gen.sql"); err != nil {
					return fmt.Errorf("remove sql generated files error: %w", err)
				}
			}

			for _, p := range s.config.Sqlc.GetPaths().OutPaths {
				resolvedP := p
				if !filepath.IsAbs(resolvedP) {
					resolvedP = filepath.Join(sqlcConfigDir, resolvedP)
				}
				if err := utils.RemoveFiles(resolvedP, "_gen.go"); err != nil {
					return fmt.Errorf("remove go generated files error: %w", err)
				}

				if err := utils.RemoveFiles(resolvedP, "_gen.sql.go"); err != nil {
					return fmt.Errorf("remove go generated files error: %w", err)
				}
			}
		}

		// save new files
		tableNamePaths := make(map[string]string, len(sqlData))
		for tableName, data := range sqlData {
			tableParams, ok := cfg.CrudParams.Tables[tableName]
			if !ok {
				return fmt.Errorf("can not find table params for table: %s", tableName)
			}

			if len(data) == 0 {
				continue
			}

			if tableParams.OutputDir != "" {
				tableNamePaths[tableParams.OutputDir] = tableName

				if err := s.saveFile(cfg, data, tableName, tableParams.OutputDir); err != nil {
					return fmt.Errorf("saveFile error: %w", err)
				}

				continue
			}

			for _, p := range queriesPaths {
				tableNamePaths[p] = tableName

				if err := s.saveFile(cfg, data, tableName, p); err != nil {
					return fmt.Errorf("saveFile error: %w", err)
				}
			}
		}

		s.logger.Infof("crud successfully generated in: %s", time.Since(timeStart))
	}

	return nil
}

type generateSQLForEachTableParams struct {
	outputPath string
	engine     string
}

// generateSQLForEachTable - generate sql queries for each tables
func (s *crud) generateSQLForEachTable(crudParams config.CrudParams, params []generateSQLForEachTableParams) (map[string][]byte, error) {
	result := make(map[string][]byte, len(params))

	for _, param := range params {
		// Get all tables from postgres
		tablesData, err := s.getTableMeta(param.outputPath)
		if err != nil {
			return nil, fmt.Errorf("getTableMeta error: %w", err)
		}

		if !engineType(param.engine).Valid() {
			return nil, fmt.Errorf("invalid engine type: %s", param.engine)
		}

		// headText := fmt.Sprintf("-- Code generated by pgxgen. DO NOT EDIT.\n-- versions:\n--   pgxgen %s\n\n", s.config.Pgxgen.Version)
		// builder.WriteString(headText)

		// Sort tables
		tableKeys := make([]string, 0, len(crudParams.Tables))
		for k := range crudParams.Tables {
			tableKeys = append(tableKeys, k)
		}
		sort.Strings(tableKeys)

		for _, tableName := range tableKeys {
			tableParams := crudParams.Tables[tableName]

			metaData := tablesData.getTableMetaData(tableName)
			if metaData == nil {
				return nil, fmt.Errorf("database does not exist table: %s", tableName)
			}

			// Sort methods
			methodKeys := make([]string, 0, len(tableParams.Methods))
			for k := range tableParams.Methods {
				methodKeys = append(methodKeys, k.String())
			}
			sort.Strings(methodKeys)

			builder := new(strings.Builder)

			for _, methodType := range methodKeys {
				methodParams := tableParams.Methods[config.MethodType(methodType)]

				params := processParams{
					builder,
					tableName,
					*metaData,
					methodParams,
					tableParams,
					engineType(param.engine),
				}

				var err error
				switch config.MethodType(methodType) {
				case METHOD_CREATE:
					err = s.processCreate(crudParams, params)
				case METHOD_UPDATE:
					err = s.processUpdate(crudParams, params)
				case METHOD_DELETE:
					err = s.processDelete(crudParams, params)
				case METHOD_GET:
					err = s.processGet(crudParams, params)
				case METHOD_FIND:
					err = s.processFind(crudParams, params)
				case METHOD_TOTAL:
					err = s.processTotal(crudParams, params)
				case METHOD_EXISTS:
					err = s.processExists(crudParams, params)
				}

				if err != nil {
					return nil, fmt.Errorf(fmt.Sprintf(ErrWhileProcessTemplate, methodType, tableName)+" error: %w", err)
				}
			}

			result[tableName] = []byte(builder.String())
		}
	}

	return result, nil
}

func (s *crud) getTableMeta(outputDir string) (tables, error) {
	groupData := make(tables)

	catalog, ok := s.catalogs[outputDir]
	if !ok {
		return nil, fmt.Errorf("can not find catalog for output dir: %s", outputDir)
	}

	for _, schema := range catalog.Catalog.Schemas {
		for _, table := range schema.Tables {
			tableMeta := &tableMetaData{
				columns: make([]string, len(table.Columns)),
			}

			for i, column := range table.Columns {
				tableMeta.columns[i] = column.Name
			}

			groupData[table.Name] = tableMeta
		}
	}

	return groupData, nil
}

func (s *crud) processCreate(cfg config.CrudParams, p processParams) error {
	methodName := p.methodParams.Name
	if methodName == "" {
		methodName = s.getMethodName(cfg, METHOD_CREATE, p.table)
	}

	operationType := "exec"
	if p.methodParams.Returning != "" {
		operationType = "one"
	}

	p.builder.WriteString(fmt.Sprintf("-- name: %s :%s\n", methodName, operationType))
	p.builder.WriteString("INSERT INTO ")
	p.builder.WriteString(p.table)
	p.builder.WriteString(" (")

	filteredColumns := cmnutils.FilterValues(p.metaData.columns, p.methodParams.SkipColumns)
	for index, name := range filteredColumns {
		if index > 0 && index < len(filteredColumns) {
			p.builder.WriteString(", ")
		}

		p.builder.WriteString(name)
	}
	p.builder.WriteString(")\n\tVALUES (")

	lastIndex := 1
	for index, name := range filteredColumns {
		if index > 0 && index < len(filteredColumns) {
			p.builder.WriteString(", ")
		}

		if len(p.methodParams.ColumnValues) > 0 {
			if value, ok := p.methodParams.ColumnValues[name]; ok {
				p.builder.WriteString(value)
				continue
			}
		}

		switch p.engine {
		case EngineTypePostgres:
			_, err := fmt.Fprintf(p.builder, "$%d", lastIndex)
			if err != nil {
				return err
			}
		case EngineTypeMysql, EngineTypeSqlite:
			_, err := fmt.Fprintf(p.builder, "?")
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("engine %s is not supported", p.engine)
		}

		lastIndex++
	}

	p.builder.WriteString(")")
	if p.methodParams.Returning != "" {
		p.builder.WriteString("\n\tRETURNING " + p.methodParams.Returning)
	}
	p.builder.WriteString(";\n\n")

	return nil
}

func (s *crud) processUpdate(cfg config.CrudParams, p processParams) error {
	primaryColumn, err := getPrimaryColumn(p.metaData.columns, p.table, p.tableParams.PrimaryColumn)
	if err != nil {
		return err
	}

	if primaryColumn == "" {
		return ErrUndefinedPrimaryColumn
	}

	methodName := p.methodParams.Name
	if methodName == "" {
		methodName = s.getMethodName(cfg, METHOD_UPDATE, p.table)
	}

	operationType := "exec"
	if p.methodParams.Returning != "" {
		operationType = "one"
	}

	p.builder.WriteString(fmt.Sprintf("-- name: %s :%s\n", methodName, operationType))
	p.builder.WriteString("UPDATE ")
	p.builder.WriteString(p.table)
	p.builder.WriteString("\n\tSET ")

	lastIndex := 1
	filteredColumns := cmnutils.FilterValues(p.metaData.columns, p.methodParams.SkipColumns)
	for index, name := range filteredColumns {
		if index > 0 && index < len(filteredColumns) {
			p.builder.WriteString(", ")
			if len(p.metaData.columns) > columnsPerLineThreshold && index%columnsPerLineThreshold == 0 {
				p.builder.WriteString("\n\t\t")
			}
		}

		if len(p.methodParams.ColumnValues) > 0 {
			if value, ok := p.methodParams.ColumnValues[name]; ok {
				p.builder.WriteString(name + "=" + value)
				continue
			}
		}

		switch p.engine {
		case EngineTypePostgres:
			_, err := fmt.Fprintf(p.builder, "%s=$%d", name, lastIndex)
			if err != nil {
				return err
			}
		case EngineTypeMysql, EngineTypeSqlite:
			_, err := fmt.Fprintf(p.builder, "%s=?", name)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("engine %s is not supported", p.engine)
		}

		lastIndex++
	}

	p.builder.WriteString("\n\t")
	p.methodParams.AddWhereParam(primaryColumn, config.WhereParamsItem{})

	if err := s.processWhereParam(p, METHOD_UPDATE, &lastIndex); err != nil {
		return err
	}

	if p.methodParams.Returning != "" {
		p.builder.WriteString("\n\tRETURNING " + p.methodParams.Returning)
	}
	p.builder.WriteString(";\n\n")

	return nil
}

func (s *crud) processDelete(cfg config.CrudParams, p processParams) error {
	primaryColumn, err := getPrimaryColumn(p.metaData.columns, p.table, p.tableParams.PrimaryColumn)
	if err != nil {
		return err
	}

	if primaryColumn == "" {
		return ErrUndefinedPrimaryColumn
	}

	methodName := p.methodParams.Name
	if methodName == "" {
		methodName = s.getMethodName(cfg, METHOD_DELETE, p.table)
	}

	p.builder.WriteString(fmt.Sprintf("-- name: %s :exec\n", methodName))
	p.builder.WriteString("DELETE FROM ")
	p.builder.WriteString(p.table)

	lastIndex := 1
	p.methodParams.AddWhereParam(primaryColumn, config.WhereParamsItem{})

	if err := s.processWhereParam(p, METHOD_DELETE, &lastIndex); err != nil {
		return err
	}
	p.builder.WriteString(";\n\n")

	return nil
}

func (s *crud) processGet(cfg config.CrudParams, p processParams) error {
	primaryColumn, err := getPrimaryColumn(p.metaData.columns, p.table, p.tableParams.PrimaryColumn)
	if err != nil {
		return err
	}

	if primaryColumn == "" {
		return ErrUndefinedPrimaryColumn
	}

	methodName := p.methodParams.Name
	if methodName == "" {
		methodName = s.getMethodName(cfg, METHOD_GET, p.table)
	}

	p.builder.WriteString(fmt.Sprintf("-- name: %s :one\n", methodName))
	p.builder.WriteString("SELECT * FROM ")
	p.builder.WriteString(p.table)

	lastIndex := 1
	p.methodParams.AddWhereParam(primaryColumn, config.WhereParamsItem{})

	if err := s.processWhereParam(p, METHOD_GET, &lastIndex); err != nil {
		return err
	}
	p.builder.WriteString(" LIMIT 1;\n\n")

	return nil
}

func (s *crud) processFind(cfg config.CrudParams, p processParams) error {
	methodName := p.methodParams.Name
	if methodName == "" {
		methodName = s.getMethodName(cfg, METHOD_FIND, p.table)
	}

	p.builder.WriteString(fmt.Sprintf("-- name: %s :many\n", methodName))
	p.builder.WriteString("SELECT * FROM ")
	p.builder.WriteString(p.table)
	lastIndex := 1
	if err := s.processWhereParam(p, METHOD_FIND, &lastIndex); err != nil {
		return err
	}
	if order := getOrderByParams(p.methodParams); order != nil {
		p.builder.WriteString(fmt.Sprintf(" ORDER BY %s %s", order.By, order.Direction))
	}
	if p.methodParams.Limit {
		switch p.engine {
		case EngineTypePostgres:
			p.builder.WriteString(fmt.Sprintf(" LIMIT $%d OFFSET $%d", lastIndex, lastIndex+1))
		case EngineTypeMysql, EngineTypeSqlite:
			p.builder.WriteString(" LIMIT ? OFFSET ?")
		default:
			return fmt.Errorf("engine %s is not supported", p.engine)
		}
	}
	p.builder.WriteString(";\n\n")
	return nil
}

func (s *crud) processTotal(cfg config.CrudParams, p processParams) error {
	methodName := p.methodParams.Name
	if methodName == "" {
		methodName = s.getMethodName(cfg, METHOD_TOTAL, p.table)
	}

	p.builder.WriteString(fmt.Sprintf("-- name: %s :one\n", methodName))
	p.builder.WriteString("SELECT count(1) as total FROM ")
	p.builder.WriteString(p.table)
	lastIndex := 1
	if err := s.processWhereParam(p, METHOD_TOTAL, &lastIndex); err != nil {
		return err
	}
	p.builder.WriteString(";\n\n")

	return nil
}

func (s *crud) processExists(cfg config.CrudParams, p processParams) error {
	methodName := p.methodParams.Name
	if methodName == "" {
		methodName = s.getMethodName(cfg, METHOD_EXISTS, p.table)
	}

	p.builder.WriteString(fmt.Sprintf("-- name: %s :one\n", methodName))
	p.builder.WriteString("SELECT EXISTS (SELECT 1 FROM ")
	p.builder.WriteString(p.table)
	lastIndex := 1
	if err := s.processWhereParam(p, METHOD_EXISTS, &lastIndex); err != nil {
		return err
	}
	switch p.engine {
	case EngineTypePostgres:
		p.builder.WriteString(" LIMIT 1)::boolean;\n\n")
	case EngineTypeMysql, EngineTypeSqlite:
		p.builder.WriteString(" LIMIT 1);\n\n")
	default:
		return fmt.Errorf("engine %s is not supported", p.engine)
	}

	return nil
}

func (s *crud) processWhereParam(p processParams, method config.MethodType, lastIndex *int) error {
	// process where params
	if params := getWhereParams(p.methodParams, method); len(params) > 0 {
		firstIter := true
		for _, wp := range params {
			if !slices.Contains(p.metaData.columns, wp.Name) {
				return fmt.Errorf("param %s does not exist in table %s", wp.Name, p.table)
			}

			if firstIter {
				strPattern := " "
				if method == METHOD_UPDATE {
					strPattern = ""
				}

				p.builder.WriteString(strPattern + "WHERE ")
			} else {
				p.builder.WriteString(" AND ")
			}

			if wp.Item.Value == "" {
				operator := wp.Item.Operator
				if operator == "" {
					operator = "="
				}

				switch p.engine {
				case EngineTypePostgres:
					_, err := fmt.Fprintf(p.builder, "%s%s$%d", wp.Name, operator, *lastIndex)
					if err != nil {
						return err
					}
				case EngineTypeMysql, EngineTypeSqlite:
					_, err := fmt.Fprintf(p.builder, "%s%s?", wp.Name, operator)
					if err != nil {
						return err
					}
				default:
					return fmt.Errorf("engine %s is not supported", p.engine)
				}

				*lastIndex++
			} else {
				p.builder.WriteString(wp.Name)
				if wp.Item.Operator != "" {
					p.builder.WriteString(fmt.Sprintf(" %s", wp.Item.Operator))
				}
				p.builder.WriteString(fmt.Sprintf(" %s", wp.Item.Value))
			}
			firstIter = false
		}
	}

	// process where additional params
	if params := getWhereAddtitionalParams(p.methodParams, method); len(params) > 0 {
		whereParamsLen := len(getWhereParams(p.methodParams, method))

		for paramIndex, param := range params {
			if paramIndex == 0 && whereParamsLen == 0 {
				p.builder.WriteString(" WHERE ")
			} else {
				strPattern := " "
				if paramIndex > 0 {
					strPattern = "\t"
				}
				p.builder.WriteString(strPattern + "AND ")
			}

			strPattern := "%s"
			if paramIndex < len(params)-1 {
				strPattern += "\n"
			}

			p.builder.WriteString(fmt.Sprintf(strPattern, param))
		}
	}

	return nil
}

func (s *crud) saveFile(cfg config.PgxgenSqlc, data []byte, tableName, path string) error {
	tableParams, ok := cfg.CrudParams.Tables[tableName]
	if !ok {
		return fmt.Errorf("can not find table params for table: %s", tableName)
	}

	if tableParams.OutputDir != "" {
		// Compare using absolute paths to handle different base directories
		absTableOut, _ := filepath.Abs(s.resolvePathFromPgxgenDir(tableParams.OutputDir))
		absPath, _ := filepath.Abs(s.resolvePathFromPgxgenDir(path))
		if absTableOut != absPath {
			return nil
		}
	}

	if tableName == "" {
		return fmt.Errorf("empty table name")
	}

	outputDir := s.resolvePathFromPgxgenDir(path)

	fileName := fmt.Sprintf("%s_gen.sql", tableName)
	if err := utils.SaveFile(outputDir, fileName, data); err != nil {
		return fmt.Errorf("SaveFile error: %w", err)
	}

	return nil
}

// resolvePathFromPgxgenDir resolves a path: if absolute, returns as-is;
// if relative, joins with pgxgen config directory.
func (s *crud) resolvePathFromPgxgenDir(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(s.pgxgenFileDir, p)
}

// func (s *crud) getMethodParams(methodType config.MethodType, p processParams) config.Method {
// 	res := p.methodParams
// 	return res
// }

func (s *crud) getMethodName(cfg config.CrudParams, methodType config.MethodType, tableName string) string {
	var methodName string
	if cfg.ExcludeTableNameFromMethods {
		methodName = methodType.String()
	} else {
		methodName = fmt.Sprintf("%s %s", methodType.String(), tableName)
	}

	methodName = stringy.New(methodName).CamelCase().UcFirst()

	if !slices.Contains([]config.MethodType{METHOD_FIND, METHOD_TOTAL}, methodType) {
		if strings.HasSuffix(methodName, "s") {
			methodName = string(methodName[:len(methodName)-1])
		}
	}

	return methodName
}

func getPrimaryColumn(columns []string, table, column string) (string, error) {
	primaryColumn := ""
	if column != "" {
		if !slices.Contains(columns, column) {
			return primaryColumn, fmt.Errorf("table %s does not have a primary column %s", table, column)
		}
		primaryColumn = column
	}

	return primaryColumn, nil
}

func getWhereParams(method config.Method, methodType config.MethodType) []whereParam {
	if strings.ToLower(methodType.String()) == "create" {
		return nil
	}

	keys := make([]string, 0, len(method.Where))
	for k := range method.Where {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := make([]whereParam, 0, len(keys))
	for _, k := range keys {
		result = append(result, whereParam{Name: k, Item: method.Where[k]})
	}
	return result
}

func getWhereAddtitionalParams(method config.Method, methodType config.MethodType) []string {
	if strings.ToLower(methodType.String()) == "create" {
		return nil
	}
	if len(method.WhereAdditional) > 0 {
		return method.WhereAdditional
	}
	return nil
}

func getOrderByParams(method config.Method) *config.OrderParam {
	if method.Order.By == "" {
		return nil
	}

	if method.Order.Direction == "" {
		method.Order.Direction = "DESC"
	}

	return &method.Order
}
