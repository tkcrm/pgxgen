package sqlc

import (
	"testing"

	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/structs"
)

func TestReplaceImports(t *testing.T) {
	str1 := `// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.16.0
// source: queries.sql

package psqldb

import (
	"context"
	"time"

	. "github.com/test/testtest/internal/models"
    _ "github.com/test/testtest/internal/models"
    asdasd "github.com/test/testtest/internal/models"
)

const a = "b{})"
)
sdv

asdvvvvasdv {}`

	str2 := `// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.16.0
// source: queries.sql

package psqldb

import "context"

const a = "b"`

	cfg := config.SqlcModels{}

	if _, err := replaceImports(str1, cfg, structs.Structs{}); err != nil {
		t.Fatal(err)
	}

	if _, err := replaceImports(str2, cfg, structs.Structs{}); err != nil {
		t.Fatal(err)
	}
}
