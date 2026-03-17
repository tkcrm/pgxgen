package crud

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed templates
var embeddedTemplates embed.FS

// customTemplateDir is set when user provides custom templates via config.
var customTemplateDir string

// SetCustomTemplateDir sets the directory for user-provided CRUD templates.
func SetCustomTemplateDir(dir string) {
	customTemplateDir = dir
}

// getTemplate returns the template content for the given engine and method.
// It first checks the custom template dir, then falls back to embedded.
func getTemplate(engineName, methodName string) (string, error) {
	fileName := methodName + ".sql.tmpl"

	// Check custom templates first
	if customTemplateDir != "" {
		customPath := filepath.Join(customTemplateDir, engineName, fileName)
		if data, err := os.ReadFile(customPath); err == nil {
			return string(data), nil
		}
	}

	// Fall back to embedded templates
	path := fmt.Sprintf("templates/%s/%s", engineName, fileName)
	data, err := embeddedTemplates.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("template not found: %s/%s: %w", engineName, fileName, err)
	}

	return string(data), nil
}
