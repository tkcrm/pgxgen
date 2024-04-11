package sqlc

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/structs"
	"github.com/tkcrm/pgxgen/utils"
	"golang.org/x/tools/imports"
)

var replaceTypesMap map[string]string = map[string]string{
	"sql.NullInt32":   "*int32",
	"sql.NullInt64":   "*int64",
	"sql.NullInt16":   "*int16",
	"sql.NullFloat64": "*float64",
	"sql.NullFloat32": "*float32",
	"sql.NullString":  "*string",
	"sql.NullBool":    "*bool",
	"sql.NullTime":    "*time.Time",
}

func (s *sqlc) replace(path string, fn replaceFunc) error {
	file, err := utils.ReadFile(path)
	if err != nil {
		return err
	}

	result, err := fn(s.config, string(file))
	if err != nil {
		return fmt.Errorf("fn for file %s error: %w", path, err)
	}

	formatedFileContent, err := imports.Process(path, []byte(result), nil)
	if err != nil {
		return err
	}

	formattedPath := filepath.Join(path)

	if err := os.WriteFile(formattedPath, formatedFileContent, os.ModePerm); err != nil {
		return fmt.Errorf("failed to write file to path \"%s\": %w", formattedPath, err)
	}

	return nil
}

func replaceStructTypes(c config.Config, str string) (string, error) {
	for old, new := range replaceTypesMap {
		str = strings.ReplaceAll(str, old, new)
	}

	return str, nil
}

// replacePackageName - replace package name for golang file
func replacePackageName(str string, sqlcModelParam config.SqlcModels) (res string, err error) {
	outputDirPath := strings.Split(sqlcModelParam.Move.OutputDir, "/")
	if len(outputDirPath) == 0 {
		return str, nil
	}

	packageName := outputDirPath[len(outputDirPath)-1]
	if sqlcModelParam.Move.PackageName != "" {
		packageName = sqlcModelParam.Move.PackageName
	}

	r := bufio.NewReader(strings.NewReader(str))

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}

		if strings.HasPrefix(line, "package") {
			res += "package " + packageName
		} else {
			res += line
		}
	}

	return res, nil
}

func replaceImports(str string, sqlcModelParam config.SqlcModels, modelFileStructs structs.Structs) (res string, err error) {
	outputDirPath := strings.Split(sqlcModelParam.Move.OutputDir, "/")
	if len(outputDirPath) == 0 {
		return str, nil
	}

	var existsSomeModelStruct bool
	for _, item := range modelFileStructs {
		re := regexp.MustCompile(fmt.Sprintf(`(?sm)\([\[\]\*]+%s[\,\){]+`, item.Name))
		if re.MatchString(str) {
			existsSomeModelStruct = true
			break
		}

		for _, field := range item.Fields {
			re := regexp.MustCompile(fmt.Sprintf(`(?sm)\s+\w+\s+%s\s+`, field.Name))
			if re.MatchString(str) {
				existsSomeModelStruct = true
				break
			}
		}
	}
	if !existsSomeModelStruct {
		return str, nil
	}

	r := regexp.MustCompile(`import (\"\w+\")`)
	r2 := regexp.MustCompile(`(?sm)^import \(\s(([^\)]+)\s)+\)`)

	if r.MatchString(str) {
		matches := r.FindStringSubmatch(str)
		if len(matches) > 1 {
			res = r.ReplaceAllString(str, getNewImports(matches[1:], sqlcModelParam.Move.PackagePath))
		}
	}

	if r2.MatchString(str) {
		matches := r2.FindStringSubmatch(str)
		if len(matches) == 3 {
			packageImports := matches[2]
			re2 := regexp.MustCompile(`(\s+(.*)\n?)`)
			rs2 := re2.FindAllStringSubmatch(packageImports, -1)
			imports := make([]string, 0, len(rs2))
			for _, item := range rs2 {
				imports = append(imports, item[2])
			}

			res = r2.ReplaceAllString(str, getNewImports(imports, sqlcModelParam.Move.PackagePath))
		}
	}

	return res, nil
}

func getNewImports(existImports []string, packagePath string) string {
	existImports = append(existImports, fmt.Sprintf(". \"%s\"", packagePath))
	return fmt.Sprintf("import(\n%s\n)", strings.Join(existImports, "\n"))
}
