package sqlc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tkcrm/pgxgen/pkg/logger"
)

// TestCleanupGeneratedConfig_RemovesEmptyDir verifies that cleanup
// removes the sqlc.yaml file and the .pgxgen directory when it is
// empty after the file is gone.
func TestCleanupGeneratedConfig_RemovesEmptyDir(t *testing.T) {
	tmp := t.TempDir()
	pgxgenDir := filepath.Join(tmp, ".pgxgen")
	sqlcPath := filepath.Join(pgxgenDir, "sqlc.yaml")
	require.NoError(t, os.MkdirAll(pgxgenDir, 0o755))
	require.NoError(t, os.WriteFile(sqlcPath, []byte("version: \"2\"\n"), 0o644))

	g := NewGenerator(logger.New(), tmp)
	g.cleanupGeneratedConfig(pgxgenDir, sqlcPath)

	assert.NoFileExists(t, sqlcPath)
	assert.NoDirExists(t, pgxgenDir)
}

// TestCleanupGeneratedConfig_PreservesNonEmptyDir verifies that the
// .pgxgen directory is kept when it contains user-created siblings
// (e.g. custom templates). Only the generated sqlc.yaml is removed.
func TestCleanupGeneratedConfig_PreservesNonEmptyDir(t *testing.T) {
	tmp := t.TempDir()
	pgxgenDir := filepath.Join(tmp, ".pgxgen")
	sqlcPath := filepath.Join(pgxgenDir, "sqlc.yaml")
	templatesDir := filepath.Join(pgxgenDir, "templates")
	require.NoError(t, os.MkdirAll(templatesDir, 0o755))
	require.NoError(t, os.WriteFile(sqlcPath, []byte("version: \"2\"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(templatesDir, "custom.tmpl"), []byte("x"), 0o644))

	g := NewGenerator(logger.New(), tmp)
	g.cleanupGeneratedConfig(pgxgenDir, sqlcPath)

	assert.NoFileExists(t, sqlcPath, "generated sqlc.yaml must be removed")
	assert.DirExists(t, pgxgenDir, ".pgxgen dir must be kept when it has other content")
	assert.DirExists(t, templatesDir, "user templates must be preserved")
}

// TestCleanupGeneratedConfig_MissingFileIsNoop verifies that cleanup
// does not error when the sqlc.yaml file was never written (e.g.
// an early error aborted Run before WriteFile).
func TestCleanupGeneratedConfig_MissingFileIsNoop(t *testing.T) {
	tmp := t.TempDir()
	pgxgenDir := filepath.Join(tmp, ".pgxgen")
	sqlcPath := filepath.Join(pgxgenDir, "sqlc.yaml")
	require.NoError(t, os.MkdirAll(pgxgenDir, 0o755))

	g := NewGenerator(logger.New(), tmp)
	g.cleanupGeneratedConfig(pgxgenDir, sqlcPath)

	assert.NoDirExists(t, pgxgenDir, "empty .pgxgen must still be removed")
}
