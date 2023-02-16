package sqlc

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/generator"
	"github.com/tkcrm/pgxgen/internal/goconstatnts"
	"github.com/tkcrm/pgxgen/internal/structs"
	sqlcpkg "github.com/tkcrm/pgxgen/pkg/sqlc"
	"github.com/tkcrm/pgxgen/utils"
	"golang.org/x/tools/imports"
)

type sqlc struct {
	config      config.Config
	goConstants goconstatnts.IGoConstants
}

func New(cfg config.Config) generator.IGenerator {
	return &sqlc{
		config:      cfg,
		goConstants: goconstatnts.New(cfg),
	}
}

func (s *sqlc) Generate(args []string) error {
	if err := s.process(args); err != nil {
		return errors.Wrap(err, "failed to generate sqlc")
	}

	fmt.Println("sqlc successfully generated")

	return nil
}

func (s *sqlc) process(args []string) error {
	if len(args) > 1 {
		args = args[1:]
	}

	// generate sqlc code
	genResult := sqlcpkg.Run(args)
	if genResult != 0 {
		return nil
	}

	if s.config.Sqlc.Version > 2 || s.config.Sqlc.Version < 1 {
		return fmt.Errorf("unsupported sqlc version: %d", s.config.Sqlc.Version)
	}

	var modelFileStructs structs.Structs
	modelPaths := s.config.Sqlc.GetPaths().ModelsPaths
	for _, path := range modelPaths {
		if err := s.processModelFilePaths(path, &modelFileStructs); err != nil {
			return errors.Wrapf(err, "processModelFilePaths error: failed to process file \"%s\"", path)
		}

		if err := s.processGoFilePaths(path, modelFileStructs); err != nil {
			return errors.Wrapf(err, "processGoFilePaths error: failed to process file \"%s\"", path)
		}
	}

	if err := s.goConstants.GenerateConstants(); err != nil {
		return err
	}

	return nil
}

func (s *sqlc) processModelFilePaths(modelFilePath string, modelFileStructs *structs.Structs) error {
	modelFileDir := filepath.Dir(modelFilePath)
	modelFileName := filepath.Base(modelFilePath)

	files, err := os.ReadDir(modelFileDir)
	if err != nil {
		return errors.Wrapf(err, "failed to read path \"%s\"", modelFileDir)
	}

	for _, file := range files {
		goFileRegexp := regexp.MustCompile(`(\.go)`)

		// skip not golang files
		if !goFileRegexp.MatchString(file.Name()) {
			continue
		}

		// replace nullable types
		if !s.config.Pgxgen.DisableAutoReaplceSqlcNullableTypes {
			if err := s.replace(filepath.Join(modelFileDir, file.Name()), replaceStructTypes); err != nil {
				return err
			}
		}

		// process models file
		if file.Name() == modelFileName {
			modelFilePath := filepath.Join(modelFileDir, file.Name())

			modelFile, err := utils.ReadFile(modelFilePath)
			if err != nil {
				return err
			}

			*modelFileStructs = structs.GetStructs(string(modelFile))

			// replace json tags
			if err := s.replace(modelFilePath, replaceJsonTags); err != nil {
				return err
			}

			// replace package name
			if err := s.replace(modelFilePath, replacePackageName); err != nil {
				return err
			}

			if s.config.Pgxgen.SqlcModels.OutputDir != "" {
				currentDir, err := os.Getwd()
				if err != nil {
					return err
				}

				newPathDir := filepath.Join(currentDir, s.config.Pgxgen.SqlcModels.OutputDir)
				oldPathDir := filepath.Join(currentDir, modelFilePath)

				// create dir if new path not exists
				if err := utils.CreatePath(newPathDir); err != nil {
					return err
				}

				// move file to new directory
				fileName := file.Name()
				if s.config.Pgxgen.SqlcModels.OutputFilename != "" {
					fileName = s.config.Pgxgen.SqlcModels.OutputFilename
				}

				newPathFile := filepath.Join(newPathDir, fileName)
				if err := os.Rename(oldPathDir, newPathFile); err != nil {
					return errors.Wrapf(err,
						"failed to move file from %s to %s",
						modelFilePath,
						newPathFile,
					)
				}
			}
		}
	}

	return nil
}

func (s *sqlc) processGoFilePaths(path string, modelFileStructs structs.Structs) error {
	modelFileDir := filepath.Dir(path)

	files, err := os.ReadDir(modelFileDir)
	if err != nil {
		return errors.Wrapf(err, "failed to read path \"%s\"", modelFileDir)
	}

	for _, file := range files {
		goFileRegexp := regexp.MustCompile(`(\.go)`)

		// skip not golang files
		if !goFileRegexp.MatchString(file.Name()) {
			continue
		}

		// replace imports
		if strings.HasSuffix(file.Name(), ".sql.go") || file.Name() == "querier.go" {
			if err := s.replace(filepath.Join(modelFileDir, file.Name()), func(c config.Config, str string) string {
				return replaceImports(c, str, modelFileStructs)
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

func replaceStructTypes(c config.Config, str string) string {
	res := str
	for old, new := range types {
		res = strings.ReplaceAll(res, old, new)
	}

	return res
}

func replaceJsonTags(c config.Config, str string) string {
	res := str
	for _, field := range c.Pgxgen.JsonTags.Omitempty {
		res = strings.ReplaceAll(res, fmt.Sprintf("json:\"%s\"", field), fmt.Sprintf("json:\"%s,omitempty\"", field))
	}

	for _, field := range c.Pgxgen.JsonTags.Hide {
		res = strings.ReplaceAll(res, fmt.Sprintf("json:\"%s\"", field), "json:\"-\"")
	}

	return res
}

func (s *sqlc) replace(path string, fn func(c config.Config, str string) string) error {
	file, err := utils.ReadFile(path)
	if err != nil {
		return err
	}

	result := fn(s.config, string(file))

	formatedFileContent, err := imports.Process(path, []byte(result), nil)
	if err != nil {
		return err
	}

	formattedPath := filepath.Join(path)

	if err := os.WriteFile(formattedPath, formatedFileContent, os.ModePerm); err != nil {
		return errors.Wrapf(err, "failed to write file to path \"%s\"", formattedPath)
	}

	return nil
}
