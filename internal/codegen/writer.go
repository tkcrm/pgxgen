package codegen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// WriteResults writes generation results to disk.
// If dryRun is true, prints diffs to stdout without writing.
func WriteResults(results []GenerationResult, dryRun bool) error {
	for _, r := range results {
		if r.Action == ActionUnchanged {
			continue
		}

		if dryRun {
			printDiff(r)
			continue
		}

		switch r.Action {
		case ActionCreate, ActionUpdate:
			dir := filepath.Dir(r.Path)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("create dir %s: %w", dir, err)
			}
			if err := os.WriteFile(r.Path, r.Content, 0o644); err != nil {
				return fmt.Errorf("write file %s: %w", r.Path, err)
			}
		case ActionDelete:
			if err := os.Remove(r.Path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("delete file %s: %w", r.Path, err)
			}
		}
	}

	return nil
}

// PrepareResult creates a GenerationResult by comparing new content with existing file.
func PrepareResult(path string, content []byte) GenerationResult {
	existing, err := os.ReadFile(path)
	if err != nil {
		// File doesn't exist — create
		return GenerationResult{
			Path:    path,
			Content: content,
			Action:  ActionCreate,
		}
	}

	if bytes.Equal(existing, content) {
		return GenerationResult{
			Path:     path,
			Content:  content,
			Existing: existing,
			Action:   ActionUnchanged,
		}
	}

	return GenerationResult{
		Path:     path,
		Content:  content,
		Existing: existing,
		Action:   ActionUpdate,
	}
}

func printDiff(r GenerationResult) {
	switch r.Action {
	case ActionCreate:
		fmt.Printf("+++ %s (new file, %d bytes)\n", r.Path, len(r.Content))
	case ActionUpdate:
		fmt.Printf("~~~ %s (modified, %d → %d bytes)\n", r.Path, len(r.Existing), len(r.Content))
	case ActionDelete:
		fmt.Printf("--- %s (deleted)\n", r.Path)
	}
}
