package utils_test

import (
	"os"
	"testing"

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
