package utils

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/tools/imports"
)

func GetGoPackageNameForFile(path, fileName string) (string, error) {
	filePath := filepath.Join(path, fileName)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	r := bufio.NewReader(bytes.NewReader(data))

	re := regexp.MustCompile(`^package (\w+)$`)
	var res string
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}

		findedStrings := re.FindStringSubmatch(strings.TrimSpace(line))
		if len(findedStrings) == 2 {
			res = findedStrings[1]
			break
		}
	}

	if res == "" {
		return "", fmt.Errorf("undefined go package name for file path: %s", filePath)
	}

	return res, nil
}

func GetGoPackageNameForDir(path string) (string, error) {
	dirData, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}

	var res string
	for _, item := range dirData {
		if item.IsDir() || !strings.HasSuffix(item.Name(), ".go") {
			continue
		}

		goPackage, err := GetGoPackageNameForFile(path, item.Name())
		if err != nil {
			return "", errors.Wrap(err, "GetGoPackageNameForFile error")
		}

		if strings.HasSuffix(goPackage, "_test") {
			continue
		}

		res = goPackage
		break
	}

	if res == "" {
		return "", fmt.Errorf("undefined go package name for path: %s", path)
	}

	return res, nil
}

func UpdateGoImports(data []byte) ([]byte, error) {
	return imports.Process("", data, nil)
}
