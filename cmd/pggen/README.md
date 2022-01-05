# pgxgen

Package `"github.com/tkcrm/pgxgen/cmd/pgxgen"` contains the command line
tool for invoking the `pgxgen` library. It generates type safe SQL
database call shims based on the objects stored in a postgres database.
This allows you to define the schema for your database objects only once,
and in the language most natural for working with relational data: SQL.

See `$CODE/go/lib/pgxgen/README.md` for more details on the features
and configuration for pgxgen.
