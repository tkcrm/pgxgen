package watcher

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/tkcrm/pgxgen/internal/codegen"
	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/pkg/logger"
)

const debounceDelay = 500 * time.Millisecond

// Watch watches schema directories and regenerates on changes.
func Watch(ctx context.Context, l logger.Logger, cfg *config.V2Config, configPath string) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer func() { _ = w.Close() }()

	configDir := filepath.Dir(configPath)

	// Add schema dirs to watcher
	for _, schema := range cfg.Schemas {
		schemaDir := schema.SchemaDir
		if !filepath.IsAbs(schemaDir) {
			schemaDir = filepath.Join(configDir, schemaDir)
		}

		if info, err := os.Stat(schemaDir); err == nil && info.IsDir() {
			if err := w.Add(schemaDir); err != nil {
				return err
			}
			l.Infof("watching %s", schemaDir)
		}
	}

	l.Info("watching for changes... (Ctrl+C to stop)")

	var debounceTimer *time.Timer

	for {
		select {
		case <-ctx.Done():
			l.Info("watcher stopped")
			return nil
		case event, ok := <-w.Events:
			if !ok {
				return nil
			}
			if !isRelevantEvent(event) {
				continue
			}

			// Debounce: reset timer on each event
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(debounceDelay, func() {
				l.Infof("change detected: %s, regenerating...", filepath.Base(event.Name))
				orch := codegen.NewOrchestrator(l, cfg, configPath)
				results, err := orch.Generate(ctx, codegen.GenerateOpts{})
				if err != nil {
					l.Infof("generate error: %s", err)
					return
				}
				if err := codegen.WriteResults(results, false); err != nil {
					l.Infof("write error: %s", err)
					return
				}
				l.Info("regeneration complete")
			})

		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			l.Infof("watcher error: %s", err)
		}
	}
}

func isRelevantEvent(event fsnotify.Event) bool {
	if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) == 0 {
		return false
	}
	ext := filepath.Ext(event.Name)
	return ext == ".sql"
}
