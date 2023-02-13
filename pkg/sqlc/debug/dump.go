package debug

import (
	"os"

	"github.com/davecgh/go-spew/spew"

	"github.com/tkcrm/pgxgen/pkg/sqlc/opts"
)

var Active bool
var Debug opts.Debug

func init() {
	Active = os.Getenv("SQLCDEBUG") != ""
	if Active {
		Debug = opts.DebugFromEnv()
	}
}

func Dump(n ...interface{}) {
	if Active {
		spew.Dump(n)
	}
}
