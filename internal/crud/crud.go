package crud

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gobeam/stringy"
	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/generator"
	"github.com/tkcrm/pgxgen/utils"
)

type crud struct {
	config config.Config
	result []byte
}

func New(cfg config.Config) generator.IGenerator {
	return &crud{
		config: cfg,
	}
}

func (s *crud) Generate(args []string) error {

	if err := s.process(args); err != nil {
		return err
	}

	for _, p := range s.config.Sqlc.Packages {
		file_name := s.config.Pgxgen.OutputCrudSqlFileName
		if file_name == "" {
			file_name = "crud_queries.sql"
		}

		if err := os.WriteFile(filepath.Join(p.Queries, file_name), s.result, 0644); err != nil {
			return err
		}
	}

	fmt.Println("crud successfully generated")

	return nil
}

func (s *crud) process(args []string) error {

	var connString string
	mySet := flag.NewFlagSet("", flag.ExitOnError)
	mySet.StringVar(&connString, "c", "", "PostgreSQL connection link")
	mySet.Parse(args[1:])

	if connString == "" {
		return errors.New("undefined PostgreSQL connection link. use flag -c")
	}

	// Get all tables from postgres
	tablesData, err := s.getTableMeta(connString)
	if err != nil {
		return err
	}

	builder := new(strings.Builder)

	// Sort tables
	tableKeys := make([]string, 0, len(s.config.Pgxgen.CrudParams.Tables))
	for k := range s.config.Pgxgen.CrudParams.Tables {
		tableKeys = append(tableKeys, k)
	}
	sort.Strings(tableKeys)

	for _, tableName := range tableKeys {

		tableParams := s.config.Pgxgen.CrudParams.Tables[tableName]

		metaData := tablesData.getTableMetaData(tableName)
		if metaData == nil {
			return fmt.Errorf("database does not exist table: \"%s\"", tableName)
		}

		// Sort methods
		methodKeys := make([]string, 0, len(tableParams.Methods))
		for k := range tableParams.Methods {
			methodKeys = append(methodKeys, k.String())
		}
		sort.Strings(methodKeys)
		for _, methodType := range methodKeys {

			methodParams := tableParams.Methods[config.MethodType(methodType)]

			processParams := processParams{builder, tableName, *metaData, methodParams, tableParams}

			var err error
			switch config.MethodType(methodType) {
			case METHOD_CREATE:
				err = s.processCreate(processParams)
			case METHOD_UPDATE:
				err = s.processUpdate(processParams)
			case METHOD_DELETE:
				err = s.processDelete(processParams)
			case METHOD_GET:
				err = s.processGet(processParams)
			case METHOD_FIND:
				err = s.processFind(processParams)
			case METHOD_TOTAL:
				err = s.processTotal(processParams)
			}

			if err != nil {
				return errors.Wrap(err, fmt.Sprintf(ErrWhileProcessTemplate, methodType, tableName))
			}

		}
	}

	s.result = []byte(builder.String())

	return nil
}

func (s *crud) getTableMeta(connString string) (tables, error) {
	groupData := make(tables)

	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}
	defer conn.Close(context.Background())

	rows, err := conn.Query(context.Background(), `
		SELECT attrelid::regclass AS TABLE_NAME,
			attname AS COLUMN_NAME
		FROM pg_attribute
		INNER JOIN pg_class ON pg_class.oid = attrelid
		WHERE attrelid IN
			(SELECT pg_class.oid
			FROM pg_class
			INNER JOIN pg_namespace ON pg_namespace.oid = pg_class.relnamespace
			WHERE pg_namespace.nspname IN ('public')
			AND pg_class.relkind IN ('r', 't'))
		AND attnum > 0
		AND attisdropped IS FALSE
		ORDER  BY pg_class.relname,
					pg_attribute.attnum
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []tableColumns{}
	for rows.Next() {
		var i tableColumns
		if err := rows.Scan(
			&i.TableName,
			&i.ColumnName,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, item := range items {
		metaData, ok := groupData[item.TableName]
		if !ok {
			groupData[item.TableName] = &tableMetaData{
				columns: []string{item.ColumnName},
			}
		} else {
			groupData[item.TableName].columns = append(metaData.columns, item.ColumnName)
		}
	}

	return groupData, nil
}

func (s *crud) processCreate(p processParams) error {

	methodName := p.methodParams.Name
	if methodName == "" {
		methodName = getMethodName(METHOD_CREATE, p.table)
	}

	operationType := "exec"
	if p.methodParams.Returning != "" {
		operationType = "one"
	}

	p.builder.WriteString(fmt.Sprintf("-- name: %s :%s\n", methodName, operationType))
	p.builder.WriteString("INSERT INTO ")
	p.builder.WriteString(p.table)
	p.builder.WriteString(" (")

	filteredColumns := utils.FilterString(p.metaData.columns, p.methodParams.SkipColumns)
	for index, name := range filteredColumns {
		if index > 0 && index < len(filteredColumns) {
			p.builder.WriteString(", ")
		}

		p.builder.WriteString(fmt.Sprintf("\"%s\"", name))
	}
	p.builder.WriteString(")\n\tVALUES (")

	lastIndex := 1
	for index, name := range filteredColumns {
		if index > 0 && index < len(filteredColumns) {
			p.builder.WriteString(", ")
		}

		if name == "created_at" {
			p.builder.WriteString("now()")
		} else {
			p.builder.WriteString(fmt.Sprintf("$%d", lastIndex))
			lastIndex++
		}
	}

	p.builder.WriteString(")")
	if p.methodParams.Returning != "" {
		p.builder.WriteString("\n\tRETURNING *")
	}
	p.builder.WriteString(";\n\n")

	return nil
}

func (s *crud) processUpdate(p processParams) error {

	primaryColumn, err := getPrimaryColumn(p.metaData.columns, p.table, p.tableParams.PrimaryColumn)
	if err != nil {
		return err
	}

	if primaryColumn == "" {
		return ErrUndefinedPrimaryColumn
	}

	methodName := p.methodParams.Name
	if methodName == "" {
		methodName = getMethodName(METHOD_UPDATE, p.table)
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
	filteredColumns := utils.FilterString(p.metaData.columns, p.methodParams.SkipColumns)
	for index, name := range filteredColumns {
		if index > 0 && index < len(filteredColumns) {
			p.builder.WriteString(", ")
			if len(p.metaData.columns) > 6 && index%6 == 0 {
				p.builder.WriteString("\n\t\t")
			}
		}

		if name == "updated_at" {
			p.builder.WriteString("\"updated_at\" = now()")
		} else {
			p.builder.WriteString(fmt.Sprintf("\"%s\"=$%d", name, lastIndex))
			lastIndex++
		}
	}

	p.builder.WriteString("\n\t")
	p.builder.WriteString(fmt.Sprintf("WHERE \"%s\"=$%d", primaryColumn, lastIndex))
	lastIndex++

	if err := s.processWhereParam(p, METHOD_UPDATE, &lastIndex); err != nil {
		return err
	}

	if p.methodParams.Returning != "" {
		p.builder.WriteString("\n\tRETURNING *")
	}
	p.builder.WriteString(";\n\n")

	return nil
}

func (s *crud) processDelete(p processParams) error {

	primaryColumn, err := getPrimaryColumn(p.metaData.columns, p.table, p.tableParams.PrimaryColumn)
	if err != nil {
		return err
	}

	if primaryColumn == "" {
		return ErrUndefinedPrimaryColumn
	}

	methodName := p.methodParams.Name
	if methodName == "" {
		methodName = getMethodName(METHOD_DELETE, p.table)
	}

	p.builder.WriteString(fmt.Sprintf("-- name: %s :exec\n", methodName))
	p.builder.WriteString("DELETE FROM ")
	p.builder.WriteString(p.table)

	lastIndex := 1
	p.builder.WriteString(fmt.Sprintf(" WHERE \"%s\"=$%d", primaryColumn, lastIndex))
	lastIndex++

	if err := s.processWhereParam(p, METHOD_DELETE, &lastIndex); err != nil {
		return err
	}
	p.builder.WriteString(";\n\n")

	return nil
}

func (s *crud) processGet(p processParams) error {

	primaryColumn, err := getPrimaryColumn(p.metaData.columns, p.table, p.tableParams.PrimaryColumn)
	if err != nil {
		return err
	}

	if primaryColumn == "" {
		return ErrUndefinedPrimaryColumn
	}

	methodName := p.methodParams.Name
	if methodName == "" {
		methodName = getMethodName(METHOD_GET, p.table)
	}

	p.builder.WriteString(fmt.Sprintf("-- name: %s :one\n", methodName))
	p.builder.WriteString("SELECT * FROM ")
	p.builder.WriteString(p.table)

	lastIndex := 1
	p.builder.WriteString(fmt.Sprintf(" WHERE \"%s\"=$%d", primaryColumn, lastIndex))
	lastIndex++

	if err := s.processWhereParam(p, METHOD_GET, &lastIndex); err != nil {
		return err
	}
	p.builder.WriteString(";\n\n")

	return nil
}

func (s *crud) processFind(p processParams) error {

	methodName := p.methodParams.Name
	if methodName == "" {
		methodName = getMethodName(METHOD_FIND, p.table)
	}

	p.builder.WriteString(fmt.Sprintf("-- name: %s :many\n", methodName))
	p.builder.WriteString("SELECT * FROM ")
	p.builder.WriteString(p.table)
	lastIndex := 1
	if err := s.processWhereParam(p, METHOD_FIND, &lastIndex); err != nil {
		return err
	}
	if order := getOrderByParams(p.methodParams, p.table); order != nil {
		p.builder.WriteString(fmt.Sprintf(" ORDER BY \"%s\" %s", order.By, order.Direction))
	}
	if p.methodParams.Limit {
		p.builder.WriteString(fmt.Sprintf(" LIMIT $%d OFFSET $%d", lastIndex, lastIndex+1))
	}
	p.builder.WriteString(";\n\n")
	return nil
}

func (s *crud) processTotal(p processParams) error {

	methodName := p.methodParams.Name
	if methodName == "" {
		methodName = getMethodName(METHOD_DELETE, p.table)
	}

	p.builder.WriteString(fmt.Sprintf("-- name: %s :one\n", methodName))
	p.builder.WriteString("SELECT count(*) as total FROM ")
	p.builder.WriteString(p.table)
	lastIndex := 1
	if err := s.processWhereParam(p, METHOD_TOTAL, &lastIndex); err != nil {
		return err
	}
	p.builder.WriteString(";\n\n")

	return nil
}

func (s *crud) processWhereParam(p processParams, method config.MethodType, lastIndex *int) error {
	if params := getWhereParams(p.methodParams, p.table, method); len(params) > 0 {

		// Sort params
		paramsKeys := make([]string, 0, len(params))
		for k := range params {
			paramsKeys = append(paramsKeys, k)
		}
		sort.Strings(paramsKeys)

		firstIter := true
		for _, param := range paramsKeys {
			item := params[param]

			if !utils.ExistInArray(p.metaData.columns, param) {
				return fmt.Errorf("param %s does not exist in table %s", param, p.table)
			}

			if utils.ExistInArray([]config.MethodType{METHOD_FIND, METHOD_TOTAL}, method) ||
				(p.tableParams.PrimaryColumn == "" && firstIter) {
				p.builder.WriteString(" WHERE ")
			} else {
				p.builder.WriteString(" AND ")
			}

			if item.Value == "" {
				operator := item.Operator
				if operator == "" {
					operator = "="
				}

				p.builder.WriteString(fmt.Sprintf("\"%s\" %s $%d", param, operator, *lastIndex))
				*lastIndex++
			} else {
				p.builder.WriteString(fmt.Sprintf("\"%s\"", param))
				if item.Operator != "" {
					p.builder.WriteString(fmt.Sprintf(" %s", item.Operator))
				}
				p.builder.WriteString(fmt.Sprintf(" %s", item.Value))
			}
			firstIter = false
		}
	}
	return nil
}

// func (s *crud) getMethodParams(methodType config.MethodType, p processParams) config.Method {
// 	res := p.methodParams
// 	return res
// }

func getMethodName(methodType config.MethodType, tableName string) string {
	methodName := stringy.New(fmt.Sprintf("%s %s", methodType.String(), tableName)).CamelCase()

	if !utils.ExistInArray([]config.MethodType{METHOD_FIND, METHOD_TOTAL}, methodType) {
		if strings.HasSuffix(methodName, "s") {
			methodName = string(methodName[:len(methodName)-1])
		}
	}

	return methodName
}

func getPrimaryColumn(columns []string, table, column string) (string, error) {
	primaryColumn := ""
	if column != "" {
		if !utils.ExistInArray(columns, column) {
			return primaryColumn, fmt.Errorf("table %s does not have a primary column %s", table, column)
		}
		primaryColumn = column
	}

	return primaryColumn, nil
}

func getWhereParams(method config.Method, table string, methodType config.MethodType) map[string]config.WhereParamsItem {
	params := make(map[string]config.WhereParamsItem)

	methodLower := strings.ToLower(methodType.String())

	// Skip create method
	if methodLower == "create" {
		return params
	}

	// Sort params
	keys := make([]string, 0, len(method.Where))
	for k := range method.Where {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, param := range keys {
		params[param] = method.Where[param]
	}

	return params
}

func getOrderByParams(method config.Method, table string) *config.OrderParam {

	if method.Order.By == "" {
		return nil
	}
	if method.Order.Direction == "" {
		method.Order.Direction = "DESC"
	}
	return &method.Order
}