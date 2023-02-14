package utils

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func CreatePath(path string) error {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}

	return nil
}

func ReadFile(filePath string) ([]byte, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file by path \"%s\"", filePath)
	}

	return file, nil
}

func SaveFile(path, fileName string, data []byte) error {
	// create path if not exist
	if err := CreatePath(path); err != nil {
		return errors.Wrap(err, "CreatePath error")
	}

	// save file
	if err := os.WriteFile(filepath.Join(path, fileName), data, os.ModePerm); err != nil {
		return errors.Wrap(err, "os write file error")
	}

	return nil
}
