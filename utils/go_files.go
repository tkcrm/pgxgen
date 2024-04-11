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
			return "", fmt.Errorf("GetGoPackageNameForFile error: %w", err)
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

func GetGoImportsFromFile(data string) []string {
	res := []string{}
	r := regexp.MustCompile(`import (\"\w+\")`)
	r2 := regexp.MustCompile(`(?sm)^import \(\s(([^\)]+)\s)+\)`)

	if r.MatchString(data) {
		matches := r.FindStringSubmatch(data)
		if len(matches) > 1 {
			res = matches[1:]
		}
	}

	if r2.MatchString(data) {
		matches := r2.FindStringSubmatch(data)
		if len(matches) == 3 {
			packageImports := matches[2]
			re2 := regexp.MustCompile(`(\s+(.*)\n?)`)
			rs2 := re2.FindAllStringSubmatch(packageImports, -1)
			imports := make([]string, 0, len(rs2))
			for _, item := range rs2 {
				imports = append(imports, item[2])
			}

			res = imports
		}
	}

	return res
}

func UpdateGoImports(data []byte) ([]byte, error) {
	return imports.Process("", data, nil)
}
