package sqlparser

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ResolveSchemaFiles expands a schema path (file or directory) into a sorted list of .sql files.
func ResolveSchemaFiles(schemaPath string) ([]string, error) {
	info, err := os.Stat(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("schema path %s: %w", schemaPath, err)
	}

	if !info.IsDir() {
		return []string{schemaPath}, nil
	}

	entries, err := os.ReadDir(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("read schema dir %s: %w", schemaPath, err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(e.Name()), ".sql") {
			files = append(files, filepath.Join(schemaPath, e.Name()))
		}
	}

	sort.Strings(files)
	return files, nil
}
