package pgxgen_test

import (
	"fmt"
	"testing"

	"github.com/tkcrm/pgxgen/internal/pgxgen"
)

var connString = "postgres://postgres:postgres@localhost:5432/testtable?sslmode=disable"

func Test_Start(t *testing.T) {

	tests := []struct {
		name string
		args []string
		res  bool
	}{
		{
			name: "empty",
			args: []string{},
			res:  true,
		},
		{
			name: "gencrud",
			args: []string{"gencrud", fmt.Sprintf("-c=%s", connString)},
			res:  true,
		},
	}

	for _, ts := range tests {
		if err := pgxgen.Start(ts.args); err != nil {
			t.Fatalf("%s faild with error: %s", ts.name, err.Error())
		}
	}
}
