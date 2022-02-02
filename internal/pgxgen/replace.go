package pgxgen

var helpMessage = `Available Commands:
  gencrud -c  Generate CRUD SQL. -c SQL_CONN_STRING
  version     Print the pgxgen version number
  
  sqlc
    compile     Statically check SQL for syntax and type errors
    completion  Generate the autocompletion script for the specified shell
    generate    Generate Go code from SQL
    help        Help about any command
    init        Create an empty sqlc.yaml settings file
    version     Print the sqlc version number

  Flags:
    -x, --experimental   enable experimental features (default: false)
    -f, --file string    specify an alternate config file (default: sqlc.yaml)
    -h, --help           help for sqlc
`

var types map[string]string = map[string]string{
	"sql.NullInt32":   "*int32",
	"sql.NullInt64":   "*int64",
	"sql.NullInt16":   "*int16",
	"sql.NullFloat64": "*float64",
	"sql.NullFloat32": "*float32",
	"pgtype.Numeric":  "*float64",
	"sql.NullString":  "*string",
	"sql.NullBool":    "*bool",
	"sql.NullTime":    "*time.Time",
	//"json.RawMessage":       "map[string]interface{}",
	//"pqtype.NullRawMessage": "map[string]interface{}",
}
