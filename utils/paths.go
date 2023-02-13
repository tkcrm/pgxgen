package utils

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func CreatePath(path string) error {
	return os.MkdirAll(path, 0755)
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
