package sqlformatter

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/generator"
	"github.com/tkcrm/pgxgen/utils"
)

type sqlformatter struct {
	config config.Config
}

func New(cfg config.Config) generator.IGenerator {
	return &sqlformatter{
		config: cfg,
	}
}

func (s *sqlformatter) Generate(args []string) error {
	for path, cfg := range s.config.Pgxgen.SQLFormatter.Paths {
		files, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		for _, file := range files {
			re := regexp.MustCompile(cfg.FileNameRegexp)
			if !re.Match([]byte(file.Name())) {
				continue
			}

			fileContent, err := utils.ReadFile(filepath.Join(path, file.Name()))
			if err != nil {
				return err
			}

			formattedFileContent := format(fileContent)
			if err != nil {
				return err
			}

			if err := utils.SaveFile(path, file.Name(), []byte(formattedFileContent)); err != nil {
				return err
			}
		}
	}

	return nil
}
