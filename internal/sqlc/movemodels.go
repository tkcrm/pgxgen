package sqlc

import (
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/utils"
)

func (s *sqlc) moveModels(
	cfg config.PgxgenSqlc,
	modelsMoved *map[string]*moveModelsData,
	files []fs.DirEntry,
	modelPath, modelFileDir, modelFileName string,
) error {
	// move model file
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}

	newPathDir := filepath.Join(currentDir, cfg.SqlcModels.Move.OutputDir)
	oldPathDir := filepath.Join(currentDir, modelPath)

	modelFileStructs, alreadyMoved := (*modelsMoved)[cfg.SchemaDir]

	if !alreadyMoved {
		modelFileData, err := utils.ReadFile(oldPathDir)
		if err != nil {
			return fmt.Errorf("failed to model read file: %w", err)
		}

		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, "", modelFileData, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("failed to parse ast of model file: %w", err)
		}

		modelFileStructs = &moveModelsData{
			fileSet:    fset,
			fileAst:    node,
			filePath:   newPathDir,
			importPath: cfg.SqlcModels.Move.PackagePath,
		}

		replacePackageName(cfg.SqlcModels, modelFileStructs)

		// create dir if new path not exists
		if err := utils.CreatePath(newPathDir); err != nil {
			return fmt.Errorf("create new dir error: %w", err)
		}

		// move file to a new directory
		fileName := modelFileName
		if cfg.SqlcModels.Move.OutputFileName != "" {
			fileName = cfg.SqlcModels.Move.OutputFileName
		}

		newPathFile := filepath.Join(newPathDir, fileName)

		// create new file
		outputFile, err := os.Create(newPathFile)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		defer outputFile.Close()

		// write file
		if err := format.Node(outputFile, fset, node); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}

		// remove old file
		if err := utils.RemoveFile(oldPathDir); err != nil {
			return fmt.Errorf("remove file %s error: %w", oldPathDir, err)
		}
	}

	for _, file := range files {
		goFileRegexp := regexp.MustCompile(`(\.go)`)

		// skip not golang files
		if !goFileRegexp.MatchString(file.Name()) {
			continue
		}

		// replace imports in generated files by sqlc
		if strings.HasSuffix(file.Name(), ".sql.go") ||
			file.Name() == "querier.go" ||
			file.Name() == "batch.go" {
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
		if err := utils.RemoveFile(oldPathDir); err != nil {
			return fmt.Errorf("remove file %s error: %w", oldPathDir, err)
		}
		return nil
	}

	(*modelsMoved)[cfg.SchemaDir] = modelFileStructs

	return nil
}
