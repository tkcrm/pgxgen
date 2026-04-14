package utils_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tkcrm/pgxgen/utils"
)

func TestSaveFile_NotExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix file permissions not applicable on windows")
	}

	dir := t.TempDir()
	nested := filepath.Join(dir, "sub")

	require.NoError(t, utils.SaveFile(nested, "generated.go", []byte("package foo\n")))

	info, err := os.Stat(filepath.Join(nested, "generated.go"))
	require.NoError(t, err)

	mode := info.Mode().Perm()
	require.Equal(t, os.FileMode(0o644), mode, "generated file must not be executable, got %o", mode)
	require.Zero(t, mode&0o111, "generated file must have no executable bits")
}
