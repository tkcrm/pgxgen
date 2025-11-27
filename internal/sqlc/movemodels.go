package sqlc

import (
	"bytes"
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
	sqlcAbsFilePath, err := filepath.Abs(s.config.ConfigPaths.SqlcConfigFilePath)
	if err != nil {
		return fmt.Errorf("failed to get sqlc config abs file path: %w", err)
	}

	sqlcDir := filepath.Dir(sqlcAbsFilePath)

	// move model file
	newPathDir := filepath.Join(sqlcDir, cfg.SqlcModels.Move.OutputDir)
	oldPathDir := filepath.Join(sqlcDir, modelPath)

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

		// write file with comments added
		var buf bytes.Buffer
		if err := format.Node(&buf, fset, node); err != nil {
			return fmt.Errorf("failed to format node: %w", err)
		}

		var output string
		if cfg.SqlcModels.IncludeStructComments {
			// Add @name comments to struct closing braces
			output = addStructCommentsToText(buf.String())
		} else {
			output = buf.String()
		}

		if _, err := outputFile.WriteString(output); err != nil {
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

// addStructCommentsToText adds @name comments to struct closing braces in text
func addStructCommentsToText(content string) string {
	// Match: type StructName struct { ... }
	// Replace closing } with } // @name StructName
	re := regexp.MustCompile(`(?s)type\s+(\w+)\s+struct\s*\{(.*?)\n\}`)
	return re.ReplaceAllString(content, `type $1 struct {$2
} // @name $1`)
}
