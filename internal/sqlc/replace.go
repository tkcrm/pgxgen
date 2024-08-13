package sqlc

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/tkcrm/pgxgen/internal/config"
	"github.com/tkcrm/pgxgen/utils"
	"golang.org/x/tools/go/ast/astutil"
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
		return fmt.Errorf("failed to read file from path \"%s\": %w", path, err)
	}

	result, err := fn(s.config, string(file))
	if err != nil {
		return fmt.Errorf("fn for file %s error: %w", path, err)
	}

	formatedFileContent, err := imports.Process(path, []byte(result), nil)
	if err != nil {
		return fmt.Errorf("failed to format file content: %w", err)
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
func replacePackageName(sqlcModelParam config.SqlcModels, modelData *moveModelsData) {
	outputDirPath := strings.Split(sqlcModelParam.Move.OutputDir, "/")
	if len(outputDirPath) == 0 {
		return
	}

	packageName := outputDirPath[len(outputDirPath)-1]
	if sqlcModelParam.Move.PackageName != "" {
		packageName = sqlcModelParam.Move.PackageName
	}

	modelData.fileAst.Name.Name = packageName
}

type moveModelsData struct {
	fileSet    *token.FileSet
	fileAst    *ast.File
	filePath   string
	importPath string
}

func replaceImports(fileBody string, sqlcModelParam config.SqlcModels, modelData *moveModelsData) (string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", fileBody, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("failed to parse file: %w", err)
	}

	for _, typeName := range modelData.extractTypes() {
		replaced := replaceTypeInAST(node, typeName, modelData.fileAst.Name.Name)
		if replaced {
			astutil.AddImport(fset, node, modelData.importPath)
		}
	}

	for _, item := range sqlcModelParam.Move.Imports {
		var addImport bool
		if item.GoType != "" {
			addImport = replaceTypeInAST(node, item.GoType, modelData.fileAst.Name.Name)
		} else {
			addImport = true
		}

		if addImport {
			astutil.AddImport(fset, node, item.Path)
		}
	}

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, node); err != nil {
		return "", fmt.Errorf("failed to print file: %w", err)
	}

	fileBody = buf.String()

	return fileBody, nil
}

func replaceTypeInAST(node *ast.File, typeName, packageAlias string) bool {
	replaced := false

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.Ident:
			if x.Name == typeName {
				x.Name = packageAlias + "." + typeName
				replaced = true
			}
		}
		return true
	})

	return replaced
}

func (s *moveModelsData) extractTypes() []string {
	var types []string

	for _, decl := range s.fileAst.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					types = append(types, typeSpec.Name.Name)
				}
			}
		}
	}

	return types
}
