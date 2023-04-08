package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
		return nil, fmt.Errorf("failed to read file by path %s: %w", filePath, err)
	}

	return file, nil
}

func SaveFile(path, fileName string, data []byte) error {
	// create path if not exist
	if err := CreatePath(path); err != nil {
		return fmt.Errorf("CreatePath error: %w", err)
	}

	// save file
	if err := os.WriteFile(filepath.Join(path, fileName), data, os.ModePerm); err != nil {
		return fmt.Errorf("os write file error: %w", err)
	}

	return nil
}

// RemoveFiles - remove all files in dir with name suffix
func RemoveFiles(dir, nameSuffix string) error {
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	dirItems, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, item := range dirItems {
		if item.IsDir() {
			continue
		}

		if strings.HasSuffix(item.Name(), nameSuffix) {
			filePath := filepath.Join(dir, item.Name())
			if err := os.Remove(filePath); err != nil {
				return err
			}
		}
	}

	return nil
}

// RemoveFile - remove file
func RemoveFile(filePath string) error {
	return os.Remove(filePath)
}
