package utils

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func ExistsPath(path string) bool {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return false
	}

	return true
}

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

func RemoveFiles(path, nameSuffix string) error {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	dirItems, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, item := range dirItems {
		if item.IsDir() {
			continue
		}

		if strings.HasSuffix(item.Name(), nameSuffix) {
			filePath := filepath.Join(path, item.Name())
			if err := os.Remove(filePath); err != nil {
				return err
			}
		}
	}

	return nil
}
