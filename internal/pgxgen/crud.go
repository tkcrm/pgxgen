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
)

type tableColumns struct {
	TableName  string
	ColumnName string
}

type filteredTables []string

func (s *filteredTables) exist(v string) bool {
	for _, i := range *s {
		if i == v {
			return true
		}
	}
	return false
}

var skipTables filteredTables = filteredTables{
	"spatial_ref_sys",
	"schema_migrations",
}

func processGenCRUD(args []string, c config.SqlcConfig) error {

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
		processCreate(builder, table, columns)
		processUpdate(builder, table, columns)
		processDelete(builder, table, columns)
		processGet(builder, table, columns)
		processFind(builder, table, columns)
		processCount(builder, table, columns)
	}

	for _, p := range c.Packages {
		if err := os.WriteFile(filepath.Join(p.Queries, "crud_queries.sql"), []byte(builder.String()), 0644); err != nil {
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

func processCreate(b *strings.Builder, table string, columns []string) {

	methodName := stringy.New(fmt.Sprintf("create %s", table)).CamelCase()

	if strings.HasSuffix(methodName, "s") {
		methodName = string(methodName[:len(methodName)-1])
	}

	b.WriteString(fmt.Sprintf("-- name: %s :one\n", methodName))
	b.WriteString("INSERT INTO ")
	b.WriteString(table)
	b.WriteString(" (")
	b.WriteString(strings.Join(columns, ", "))
	b.WriteString(")\n\tVALUES (")

	lastIndex := 1
	for index, name := range columns {
		if name == "created_at" {
			b.WriteString("now()")
		} else {
			b.WriteString(fmt.Sprintf("$%d", lastIndex))
			lastIndex++
		}
		if index != len(columns)-1 {
			b.WriteString(", ")
		}
	}
	b.WriteString(")\n\tRETURNING *;\n\n")
}

func processUpdate(b *strings.Builder, table string, columns []string) {

	methodName := stringy.New(fmt.Sprintf("update %s", table)).CamelCase()

	if strings.HasSuffix(methodName, "s") {
		methodName = string(methodName[:len(methodName)-1])
	}

	b.WriteString(fmt.Sprintf("-- name: %s :exec\n", methodName))
	b.WriteString("UPDATE ")
	b.WriteString(table)
	b.WriteString("\n\tSET ")

	lastIndex := 1
	for index, name := range columns {
		if name == "updated_at" {
			b.WriteString("updated_at = now()")
		} else {
			b.WriteString(fmt.Sprintf("%s = $%d", name, lastIndex))
			lastIndex++
		}

		if index != len(columns)-1 {
			b.WriteString(", ")
			if len(columns) > 6 && lastIndex%6 == 0 {
				b.WriteString("\n\t\t")
			}
		}
	}

	b.WriteString(fmt.Sprintf("\n\tWHERE id=$%d;\n\n", lastIndex))
}

func processDelete(b *strings.Builder, table string, columns []string) {

	methodName := stringy.New(fmt.Sprintf("delete %s", table)).CamelCase()

	if strings.HasSuffix(methodName, "s") {
		methodName = string(methodName[:len(methodName)-1])
	}

	b.WriteString(fmt.Sprintf("-- name: %s :exec\n", methodName))
	b.WriteString("DELETE FROM ")
	b.WriteString(table)
	b.WriteString(" WHERE id=$1;\n\n")
}

func processGet(b *strings.Builder, table string, columns []string) {

	methodName := stringy.New(fmt.Sprintf("get %s", table)).CamelCase()

	if strings.HasSuffix(methodName, "s") {
		methodName = string(methodName[:len(methodName)-1])
	}

	b.WriteString(fmt.Sprintf("-- name: %s :one\n", methodName))
	b.WriteString("SELECT * FROM ")
	b.WriteString(table)
	b.WriteString(" WHERE id=$1;\n\n")
}

func processFind(b *strings.Builder, table string, columns []string) {

	methodName := stringy.New(fmt.Sprintf("find %s", table)).CamelCase()

	b.WriteString(fmt.Sprintf("-- name: %s :one\n", methodName))
	b.WriteString("SELECT * FROM ")
	b.WriteString(table)
	b.WriteString(" ORDER BY id DESC LIMIT $1 OFFSET $2;\n\n")
}

func processCount(b *strings.Builder, table string, columns []string) {

	methodName := stringy.New(fmt.Sprintf("count %s", table)).CamelCase()

	b.WriteString(fmt.Sprintf("-- name: %s :one\n", methodName))
	b.WriteString("SELECT count(*) as count FROM ")
	b.WriteString(table)
	b.WriteString(";\n\n")
}
