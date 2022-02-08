package pgxgen

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gobeam/stringy"
	"github.com/jackc/pgx/v4"
	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/utils"
)

type tableColumns struct {
	TableName  string
	ColumnName string
}

type stringArr []string

func (s *stringArr) exist(v string) bool {
	for _, i := range *s {
		if i == v {
			return true
		}
	}
	return false
}

var skipTables stringArr = stringArr{
	"spatial_ref_sys",
	"schema_migrations",
}

var skipCreateColumns stringArr = stringArr{
	"id",
	"updated_at",
}

var skipUpdateColumns stringArr = stringArr{
	"id",
	"created_at",
}

type processCRUDParams struct {
	config  config.Pgxgen
	builder *strings.Builder
	table   string
	columns []string
}

func generateCRUD(args []string, c config.Config) error {

	var connString string
	mySet := flag.NewFlagSet("", flag.ExitOnError)
	mySet.StringVar(&connString, "c", "", "PostgreSQL connections link")
	mySet.Parse(args[1:])

	if connString == "" {
		return errors.New("undefined PostgreSQL connections link. use flag -c")
	}

	groupData, err := getTableMeta(connString)
	if err != nil {
		return err
	}

	builder := new(strings.Builder)
	for table, columns := range groupData {
		params := processCRUDParams{c.Pgxgen, builder, table, columns}
		if err := processCreate(params); err != nil {
			return err
		}
		if err := processUpdate(params); err != nil {
			return err
		}
		if err := processDelete(params); err != nil {
			return err
		}
		if err := processGet(params); err != nil {
			return err
		}
		if err := processFind(params); err != nil {
			return err
		}
		if err := processTotal(params); err != nil {
			return err
		}
	}

	for _, p := range c.Sqlc.Packages {
		file_name := c.Pgxgen.OutputCrudSqlFileName
		if file_name == "" {
			file_name = "crud_queries.sql"
		}

		if err := os.WriteFile(filepath.Join(p.Queries, file_name), []byte(builder.String()), 0644); err != nil {
			return err
		}
	}

	return nil
}

func getTableMeta(connString string) (map[string][]string, error) {
	groupData := make(map[string][]string)

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

		if skipTables.exist(item.TableName) {
			continue
		}

		if _, ok := groupData[item.TableName]; !ok {
			groupData[item.TableName] = []string{}
		}
		groupData[item.TableName] = append(groupData[item.TableName], item.ColumnName)
	}

	return groupData, nil
}

func processCreate(p processCRUDParams) error {

	methodName := stringy.New(fmt.Sprintf("create %s", p.table)).CamelCase()

	if strings.HasSuffix(methodName, "s") {
		methodName = string(methodName[:len(methodName)-1])
	}

	p.builder.WriteString(fmt.Sprintf("-- name: %s :one\n", methodName))
	p.builder.WriteString("INSERT INTO ")
	p.builder.WriteString(p.table)
	p.builder.WriteString(" (")
	for index, name := range p.columns {
		if index > 1 && index != len(p.columns)-1 {
			p.builder.WriteString(", ")
		}

		if skipCreateColumns.exist(name) {
			continue
		} else {
			p.builder.WriteString(fmt.Sprintf("\"%s\"", name))
		}
	}
	p.builder.WriteString(")\n\tVALUES (")

	lastIndex := 1
	for index, name := range p.columns {
		if lastIndex > 1 && index != len(p.columns)-1 {
			p.builder.WriteString(", ")
		}

		if name == "created_at" {
			p.builder.WriteString("now()")
		} else if skipCreateColumns.exist(name) {
			continue
		} else {
			p.builder.WriteString(fmt.Sprintf("$%d", lastIndex))
			lastIndex++
		}
	}
	p.builder.WriteString(")\n\tRETURNING *;\n\n")

	return nil
}

func processUpdate(p processCRUDParams) error {

	methodName := stringy.New(fmt.Sprintf("update %s", p.table)).CamelCase()

	if strings.HasSuffix(methodName, "s") {
		methodName = string(methodName[:len(methodName)-1])
	}

	p.builder.WriteString(fmt.Sprintf("-- name: %s :one\n", methodName))
	p.builder.WriteString("UPDATE ")
	p.builder.WriteString(p.table)
	p.builder.WriteString("\n\tSET ")

	lastIndex := 1
	for index, name := range p.columns {
		if lastIndex > 1 && index != len(p.columns)-1 {
			p.builder.WriteString(", ")
			if len(p.columns) > 6 && lastIndex%6 == 0 {
				p.builder.WriteString("\n\t\t")
			}
		}

		if name == "updated_at" {
			p.builder.WriteString("\"updated_at\" = now()")
		} else if skipUpdateColumns.exist(name) {
			continue
		} else {
			p.builder.WriteString(fmt.Sprintf("\"%s\"=$%d", name, lastIndex))
			lastIndex++
		}
	}

	p.builder.WriteString("\n\t")
	if err := processWhereParam(p, "f", &lastIndex); err != nil {
		return err
	}
	if lastIndex == 1 {
		p.builder.WriteString(fmt.Sprintf("WHERE id=$%d", lastIndex))
	} else {
		p.builder.WriteString(fmt.Sprintf(" AND id=$%d", lastIndex))
	}

	p.builder.WriteString("\n\tRETURNING *;\n\n")

	return nil
}

func processDelete(p processCRUDParams) error {

	methodName := stringy.New(fmt.Sprintf("delete %s", p.table)).CamelCase()

	if strings.HasSuffix(methodName, "s") {
		methodName = string(methodName[:len(methodName)-1])
	}

	p.builder.WriteString(fmt.Sprintf("-- name: %s :exec\n", methodName))
	p.builder.WriteString("DELETE FROM ")
	p.builder.WriteString(p.table)
	lastIndex := 1
	if err := processWhereParam(p, "f", &lastIndex); err != nil {
		return err
	}
	if lastIndex == 1 {
		p.builder.WriteString(fmt.Sprintf(" WHERE id=$%d;\n\n", lastIndex))
	} else {
		p.builder.WriteString(fmt.Sprintf(" AND id=$%d;\n\n", lastIndex))
	}

	return nil
}

func processGet(p processCRUDParams) error {

	methodName := stringy.New(fmt.Sprintf("get %s", p.table)).CamelCase()

	if strings.HasSuffix(methodName, "s") {
		methodName = string(methodName[:len(methodName)-1])
	}

	p.builder.WriteString(fmt.Sprintf("-- name: %s :one\n", methodName))
	p.builder.WriteString("SELECT * FROM ")
	p.builder.WriteString(p.table)
	lastIndex := 1
	if err := processWhereParam(p, "f", &lastIndex); err != nil {
		return err
	}
	if lastIndex == 1 {
		p.builder.WriteString(fmt.Sprintf(" WHERE id=$%d;\n\n", lastIndex))
	} else {
		p.builder.WriteString(fmt.Sprintf(" AND id=$%d;\n\n", lastIndex))
	}

	return nil
}

func processFind(p processCRUDParams) error {

	methodName := stringy.New(fmt.Sprintf("find %s", p.table)).CamelCase()

	p.builder.WriteString(fmt.Sprintf("-- name: %s :many\n", methodName))
	p.builder.WriteString("SELECT * FROM ")
	p.builder.WriteString(p.table)
	lastIndex := 1
	if err := processWhereParam(p, "f", &lastIndex); err != nil {
		return err
	}
	if orderBy := p.config.GetOrderByParams(p.table); orderBy != nil {
		p.builder.WriteString(fmt.Sprintf(" ORDER BY %s %s", orderBy.By, orderBy.Order))
	}
	if p.config.GetLimitParam(p.table) {
		p.builder.WriteString(fmt.Sprintf(" LIMIT $%d OFFSET $%d", lastIndex, lastIndex+1))
	}
	p.builder.WriteString(";\n\n")
	return nil
}

func processTotal(p processCRUDParams) error {

	methodName := stringy.New(fmt.Sprintf("total %s", p.table)).CamelCase()

	p.builder.WriteString(fmt.Sprintf("-- name: %s :one\n", methodName))
	p.builder.WriteString("SELECT count(*) as total FROM ")
	p.builder.WriteString(p.table)
	lastIndex := 1
	if err := processWhereParam(p, "t", &lastIndex); err != nil {
		return err
	}
	p.builder.WriteString(";\n\n")

	return nil
}

func processWhereParam(p processCRUDParams, method string, lastIndex *int) error {
	if params := p.config.GetWhereParams(p.table, method); len(params) > 0 {
		for index, param := range params {
			if !utils.ExistInStringArray(p.columns, param) {
				return fmt.Errorf("param %s does not exist in table %s", param, p.table)
			}
			if index == 0 {
				p.builder.WriteString(" WHERE ")
			} else {
				p.builder.WriteString(" AND ")
			}
			p.builder.WriteString(fmt.Sprintf("%s=$%d", param, *lastIndex))
			*lastIndex++
		}
	}
	return nil
}
