package sqlc

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	sqlccli "github.com/kyleconroy/sqlc/pkg/cli"
	"github.com/pkg/errors"
	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/generator"
	"golang.org/x/tools/imports"
)

type sqlc struct {
	config config.Config
}

func New(cfg config.Config) generator.IGenerator {
	return &sqlc{
		config: cfg,
	}
}

func (s *sqlc) Generate(args []string) error {
	if err := s.process(args); err != nil {
		return err
	}

	fmt.Println("sqlc successfully generated")

	return nil
}

func (s *sqlc) process(args []string) error {
	if len(args) > 1 {
		args = args[1:]
	}

	genResult := sqlccli.Run(args)
	if genResult != 0 {
		return nil
	}

	if s.config.Sqlc.Version > 2 && s.config.Sqlc.Version < 1 {
		return nil
	}

	if s.config.Sqlc.Version == 1 {
		for _, p := range s.config.Sqlc.Packages {
			modelFileName := p.OutputModelsFileName
			if modelFileName == "" {
				modelFileName = "models.go"
			}

			if err := s.processFile(p.Path, modelFileName); err != nil {
				return errors.Wrapf(err, "failed to process file \"%s\"", modelFileName)
			}
		}
	}

	if s.config.Sqlc.Version == 2 {
		for _, p := range s.config.Sqlc.SQL {
			modelFileName := p.Gen.Go.OutputModelsFileName
			if modelFileName == "" {
				modelFileName = "models.go"
			}

			if err := s.processFile(p.Gen.Go.Out, modelFileName); err != nil {
				return errors.Wrapf(err, "failed to process file")
			}
		}
	}

	return nil
}

func (s *sqlc) processFile(path, modelFile string) error {
	files, err := os.ReadDir(path)
	if err != nil {
		return errors.Wrapf(err, "failed to read path \"%s\"", path)
	}

	for _, file := range files {
		r := regexp.MustCompile(`(\.go)`)

		if !s.config.Pgxgen.DisableAutoReaplceSqlcNullableTypes {
			if r.MatchString(file.Name()) {
				if err := s.replace(filepath.Join(path, file.Name()), replaceStructTypes); err != nil {
					return errors.Wrap(err, "replace 1 error")
				}
			}
		}

		if file.Name() == modelFile {
			if err := s.replace(filepath.Join(path, file.Name()), replaceJsonTags); err != nil {
				return errors.Wrap(err, "replace 2 error")
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
	file, err := os.ReadFile(path)
	if err != nil {
		return errors.Wrapf(err, "failed to read path \"%s\"", path)
	}

	result := fn(s.config, string(file))

	formated, err := imports.Process(path, []byte(result), nil)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(path), formated, os.ModePerm); err != nil {
		return errors.Wrapf(err, "failed to write file to path \"%s\"", filepath.Join(path))
	}

	return nil
}
