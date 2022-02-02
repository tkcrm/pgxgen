package pgxgen_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/pgxgen"
	"gopkg.in/yaml.v3"
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

func Test_Config(t *testing.T) {
	var c config.PgxgenConfig

	configFile, err := os.ReadFile("../../pgxgen.yaml")
	if err != nil {
		t.Fatal(err)
	}

	if err := yaml.Unmarshal(configFile, &c); err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%+v", c)
}
