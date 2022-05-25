package sqlc

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	sqlccli "github.com/kyleconroy/sqlc/pkg/cli"
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

	for _, p := range s.config.Sqlc.Packages {

		files, err := os.ReadDir(p.Path)
		if err != nil {
			return err
		}

		modelFileName := p.OutputModelsFileName
		if modelFileName == "" {
			modelFileName = "models.go"
		}

		for _, file := range files {
			r := regexp.MustCompile(`(\.go)`)
			if r.MatchString(file.Name()) {
				if err := s.replace(filepath.Join(p.Path, file.Name()), replaceStructTypes); err != nil {
					return err
				}
			}

			if file.Name() == modelFileName {
				if err := s.replace(filepath.Join(p.Path, file.Name()), replaceJsonTags); err != nil {
					return err
				}
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
		return err
	}

	result := fn(s.config, string(file))

	formated, err := imports.Process(path, []byte(result), nil)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(path), formated, os.ModePerm); err != nil {
		return err
	}

	return nil
}
