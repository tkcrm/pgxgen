package pgxgen

var helpMessage = `Available Commands:
  crud -c  Generate CRUD SQL. -c SQL_CONN_STRING
  models   Generate models with aditional params based on sql models
  ts       Generate typescript code based on go structs
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
