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
	"github.com/tkcrm/pgxgen/pkg/logger"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/imports"
)

// PostProcess handles:
// 1. Removing models.go from each sqlc output dir
// 2. Replacing bare type references with qualified imports (User → models.User)
func PostProcess(l logger.Logger, configDir string, schema *config.SchemaConfig, cfg *sqlcConfig) error {
	if schema.Models == nil {
		return nil
	}

	modelsPkgPath := schema.Models.PackagePath
	modelsPkgName := schema.Models.PackageName
	if modelsPkgName == "" {
		modelsPkgName = "models"
	}

	// Parse models file to extract type names
	modelsDir := schema.Models.OutputDir
	if !filepath.IsAbs(modelsDir) {
		modelsDir = filepath.Join(configDir, modelsDir)
	}
	modelsFile := filepath.Join(modelsDir, schema.Models.GetOutputFileName())

	var modelTypes []string
	if _, err := os.Stat(modelsFile); err == nil {
		modelTypes = extractTypesFromFile(modelsFile)
	}

	// Add custom types
	modelTypes = append(modelTypes, schema.Models.CustomTypes...)

	for _, entry := range cfg.SQL {
		// entry.Gen.Go.Out has "../" prefix for sqlc (relative to .pgxgen/),
		// but PostProcess runs from configDir (project root).
		// Strip the "../" prefix to get path relative to configDir.
		outputDir := stripRelPrefix(entry.Gen.Go.Out)
		if !filepath.IsAbs(outputDir) {
			outputDir = filepath.Join(configDir, outputDir)
		}

		// 1. Remove models.go
		sqlcModelsFile := filepath.Join(outputDir, "models.go")
		if _, err := os.Stat(sqlcModelsFile); err == nil {
			if err := os.Remove(sqlcModelsFile); err != nil {
				return fmt.Errorf("remove %s: %w", sqlcModelsFile, err)
			}
			l.Infof("removed sqlc models: %s", sqlcModelsFile)
		}

		// 2. Replace imports in Go files
		if modelsPkgPath != "" && len(modelTypes) > 0 {
			goFiles, _ := filepath.Glob(filepath.Join(outputDir, "*.go"))
			for _, goFile := range goFiles {
				if err := replaceImportsInFile(goFile, modelsPkgPath, modelsPkgName, modelTypes); err != nil {
					l.Infof("warning: replace imports in %s: %s", goFile, err)
				}
			}
		}

	}

	return nil
}

func extractTypesFromFile(path string) []string {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil
	}

	var types []string
	for _, decl := range node.Decls {
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

func replaceImportsInFile(filePath, pkgPath, pkgName string, typeNames []string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, data, parser.ParseComments)
	if err != nil {
		return err
	}

	changed := false
	for _, typeName := range typeNames {
		if replaceTypeInAST(node, typeName, pkgName) {
			changed = true
			astutil.AddImport(fset, node, pkgPath)
		}
	}

	if !changed {
		return nil
	}

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, node); err != nil {
		return err
	}

	formatted, err := imports.Process(filePath, buf.Bytes(), nil)
	if err != nil {
		formatted = buf.Bytes()
	}

	return os.WriteFile(filePath, formatted, 0o644)
}

func replaceTypeInAST(node *ast.File, typeName, packageAlias string) bool {
	replaced := false

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.Field:
			if replaceTypeInExpr(x.Type, typeName, packageAlias) {
				replaced = true
			}
		case *ast.FuncType:
			for _, param := range x.Params.List {
				if replaceTypeInExpr(param.Type, typeName, packageAlias) {
					replaced = true
				}
			}
			if x.Results != nil {
				for _, result := range x.Results.List {
					if replaceTypeInExpr(result.Type, typeName, packageAlias) {
						replaced = true
					}
				}
			}
		case *ast.AssignStmt:
			for _, expr := range x.Rhs {
				if replaceTypeInExpr(expr, typeName, packageAlias) {
					replaced = true
				}
			}
		case *ast.CallExpr:
			for _, arg := range x.Args {
				if replaceTypeInExpr(arg, typeName, packageAlias) {
					replaced = true
				}
			}
		case *ast.ValueSpec:
			if replaceTypeInExpr(x.Type, typeName, packageAlias) {
				replaced = true
			}
		case *ast.TypeSpec:
			if replaceTypeInExpr(x.Type, typeName, packageAlias) {
				replaced = true
			}
		}
		return true
	})

	return replaced
}

func replaceTypeInExpr(expr ast.Expr, typeName, packageAlias string) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		if e.Name == typeName {
			e.Name = packageAlias + "." + typeName
			return true
		}
	case *ast.StarExpr:
		return replaceTypeInExpr(e.X, typeName, packageAlias)
	case *ast.ArrayType:
		return replaceTypeInExpr(e.Elt, typeName, packageAlias)
	case *ast.SliceExpr:
		return replaceTypeInExpr(e.X, typeName, packageAlias)
	case *ast.CompositeLit:
		return replaceTypeInExpr(e.Type, typeName, packageAlias)
	}
	return false
}

// stripRelPrefix removes the leading "../" prefix that was added for sqlc
// path resolution (relative to .pgxgen/), returning the path relative to configDir.
func stripRelPrefix(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Clean(strings.TrimPrefix(p, "../"))
}
