package utils_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tkcrm/pgxgen/utils"
)

func Test_GetGoPackageNameForFile(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	goPackage, err := utils.GetGoPackageNameForFile(pwd, "go_files.go")
	if err != nil {
		t.Fatal(err)
	}

	if goPackage != "utils" {
		t.Fatalf("returned go package is different: %s; expected: utils", goPackage)
	}
}

func Test_GetGoPackageNameForDir(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	goPackage, err := utils.GetGoPackageNameForDir(pwd)
	if err != nil {
		t.Fatal(err)
	}

	if goPackage != "utils" {
		t.Fatalf("returned go package is different: %s; expected: utils", goPackage)
	}
}

var testImportsStr1 = `// Code generated by sqlc. DO NOT EDIT.
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

var testImportsStr2 = `// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.16.0
// source: queries.sql

package psqldb

import "context"

const a = "b"`

func Test_TestImports(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:  "multi",
			input: testImportsStr1,
			expected: []string{
				"\"context\"",
				"\"time\"",
				". \"github.com/test/testtest/internal/models\"",
				"_ \"github.com/test/testtest/internal/models\"",
				"asdasd \"github.com/test/testtest/internal/models\"",
			},
		},
		{
			name:  "single",
			input: testImportsStr2,
			expected: []string{
				"\"context\"",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := utils.GetGoImportsFromFile(tc.input)
			assert.Equal(t, tc.expected, res)
		})
	}
}
