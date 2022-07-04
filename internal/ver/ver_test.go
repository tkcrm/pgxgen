package ver_test

import (
	"fmt"
	"testing"

	version "github.com/tkcrm/pgxgen/internal/ver"
)

func Test_CheckLastestReleaseVersion(t *testing.T) {
	resp, err := version.CheckLastestReleaseVersion("v0.0.9")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(resp.Message)
}
