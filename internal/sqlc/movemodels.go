package sqlc

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/structs"
	"github.com/tkcrm/pgxgen/utils"
)

func (s *sqlc) moveModels(
	cfg config.PgxgenSqlc,
	modelsMoved *map[string]structs.Structs,
	files []fs.DirEntry,
	modelPath, modelFileDir, modelFileName string,
) error {
	modelFileStructs, alreadyMoved := (*modelsMoved)[cfg.SchemaDir]

	if !alreadyMoved {
		// get structs from model file
		modelFileStructs = structs.GetStructsByFilePath(modelPath)

		// replace package name in model file
		if err := s.replace(
			modelPath,
			func(c config.Config, str string) (string, error) {
				return replacePackageName(str, cfg.SqlcModels)
			},
		); err != nil {
			return fmt.Errorf("replacePackageName error: %w", err)
		}
	}

	for _, file := range files {
		goFileRegexp := regexp.MustCompile(`(\.go)`)

		// skip not golang files
		if !goFileRegexp.MatchString(file.Name()) {
			continue
		}

		// replace imports in generated files by sqlc
		if strings.HasSuffix(file.Name(), ".sql.go") || file.Name() == "querier.go" {
			if err := s.replace(
				filepath.Join(modelFileDir, file.Name()),
				func(c config.Config, str string) (string, error) {
					return replaceImports(str, cfg.SqlcModels, modelFileStructs)
				},
			); err != nil {
				return fmt.Errorf("replaceImports error: %w", err)
			}
		}
	}

	// delete models.go if already moved
	if alreadyMoved {
		if err := utils.RemoveFile(modelPath); err != nil {
			return fmt.Errorf("remove file %s error: %w", modelPath, err)
		}
		return nil
	}

	// move model file
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}

	newPathDir := filepath.Join(currentDir, cfg.SqlcModels.Move.OutputDir)
	oldPathDir := filepath.Join(currentDir, modelPath)

	// create dir if new path not exists
	if err := utils.CreatePath(newPathDir); err != nil {
		return fmt.Errorf("create new dir error: %w", err)
	}

	// move file to new directory
	fileName := modelFileName
	if cfg.SqlcModels.Move.OutputFileName != "" {
		fileName = cfg.SqlcModels.Move.OutputFileName
	}

	newPathFile := filepath.Join(newPathDir, fileName)
	if err := os.Rename(oldPathDir, newPathFile); err != nil {
		return fmt.Errorf(
			"failed to move file from %s to %s: %w",
			modelPath,
			newPathFile,
			err,
		)
	}

	(*modelsMoved)[cfg.SchemaDir] = modelFileStructs

	return nil
}
