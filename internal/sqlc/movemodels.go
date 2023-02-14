package sqlc

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"

	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/internal/structs"
)

// replacePackageName - replace package name for golang file
func replacePackageName(c config.Config, str string) (res string) {
	if c.Pgxgen.SqlcModels.OutputDir == "" {
		return str
	}

	outputDirPath := strings.Split(c.Pgxgen.SqlcModels.OutputDir, "/")
	if len(outputDirPath) == 0 {
		return str
	}

	packageName := outputDirPath[len(outputDirPath)-1]
	if c.Pgxgen.SqlcModels.PackageName != "" {
		packageName = c.Pgxgen.SqlcModels.PackageName
	}

	r := bufio.NewReader(strings.NewReader(str))

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}

		if strings.HasPrefix(line, "package") {
			res += "package " + packageName
		} else {
			res += line
		}
	}

	return res
}

func replaceImports(c config.Config, str string, modelFileStructs structs.Structs) (res string) {
	if c.Pgxgen.SqlcModels.OutputDir == "" {
		return str
	}

	outputDirPath := strings.Split(c.Pgxgen.SqlcModels.OutputDir, "/")
	if len(outputDirPath) == 0 {
		return str
	}

	if c.Pgxgen.SqlcModels.PackagePath == "" {
		log.Fatal("empty package path")
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
		return str
	}

	r := regexp.MustCompile(`import (\"\w+\")`)
	r2 := regexp.MustCompile(`(?sm)^import \(\s(([^\)]+)\s)+\)`)

	if r.MatchString(str) {
		matches := r.FindStringSubmatch(str)
		if len(matches) > 1 {
			res = r.ReplaceAllString(str, getNewImports(matches[1:], c.Pgxgen.SqlcModels.PackagePath))
		}
	}

	if r2.MatchString(str) {
		matches := r2.FindStringSubmatch(str)
		if len(matches) == 3 {
			packageImports := matches[2]
			var re2 = regexp.MustCompile(`(\s+(.*)\n?)`)
			rs2 := re2.FindAllStringSubmatch(packageImports, -1)
			imports := make([]string, 0, len(rs2))
			for _, item := range rs2 {
				imports = append(imports, item[2])
			}

			res = r2.ReplaceAllString(str, getNewImports(imports, c.Pgxgen.SqlcModels.PackagePath))
		}
	}

	return res
}

func getNewImports(existImports []string, packagePath string) string {
	existImports = append(existImports, fmt.Sprintf(". \"%s\"", packagePath))
	return fmt.Sprintf("import(\n%s\n)", strings.Join(existImports, "\n"))
}
