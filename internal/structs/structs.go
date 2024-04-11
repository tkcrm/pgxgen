package structs

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/gobeam/stringy"
	"github.com/tkcrm/pgxgen/utils"
	"golang.org/x/tools/go/packages"
)

type (
	Types   map[string]TypesParameters
	Structs map[string]*StructParameters
)

func (s Structs) AddStruct(name string, params *StructParameters) {
	s[name] = params
}

func (st *Structs) RemoveUnexportedFields() {
	for _, s := range *st {
		newFields := []*StructField{}
		for _, field := range s.Fields {
			if field.IsExported() {
				newFields = append(newFields, field)
			}
		}
		s.Fields = newFields
	}
}

func loadPackages(dirPath string) ([]*packages.Package, error) {
	conf := &packages.Config{
		Dir: dirPath,
		Mode: packages.NeedFiles |
			packages.NeedDeps |
			packages.NeedSyntax |
			packages.NeedImports |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedModule,
	}

	pkgs, err := packages.Load(conf)
	if err != nil {
		return nil, fmt.Errorf("load packages error: %v", err)
	}

	// if packages.PrintErrors(pkgs) > 0 {
	// 	return nil, fmt.Errorf("failed to load packages")
	// }

	return pkgs, nil
}

func GetStructsByFilePath(filePath string) Structs {
	structs := make(Structs)

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	// get file imports
	fileImports := make([]string, 0, len(node.Imports))
	for _, i := range node.Imports {
		var name, path string
		if i.Name != nil {
			name = i.Name.Name
		}
		if i.Path != nil {
			path = i.Path.Value
		}
		fileImports = append(fileImports, strings.TrimSpace(fmt.Sprintf("%s %s", name, path)))
	}

	_externalTypes := make(map[string]*ast.TypeSpec)

	// get struct types
	ast.Inspect(node, func(n ast.Node) bool {
		sp := &StructParameters{
			Imports: fileImports,
		}

		switch n := n.(type) {
		case *ast.TypeSpec:
			if _, ok := n.Type.(*ast.StructType); ok {
				if err := parseTypeSpec(node, sp, n); err != nil {
					log.Fatal(err)
				}
				if sp.Name != "" {
					structs.AddStruct(sp.Name, sp)
				}
			} else {
				_externalTypes[n.Name.Name] = n
			}

			// default:
			// 	spew.Dump(n)
		}

		return true
	})

	pkgs, err := loadPackages(filepath.Dir(filePath))
	if err != nil {
		log.Fatal(err)
	}

	for typeName, item := range _externalTypes {
		se, ok := item.Type.(*ast.SelectorExpr)
		if !ok {
			continue
		}

		x, ok := se.X.(*ast.Ident)
		if !ok {
			continue
		}

		packages.Visit(pkgs, nil, func(p *packages.Package) {
			for _, syntax := range p.Syntax {
				if syntax.Name.Name != x.Name {
					continue
				}

				for _, decl := range syntax.Decls {
					genDecl, ok := decl.(*ast.GenDecl)
					if !ok {
						continue
					}

					for _, spec := range genDecl.Specs {
						ts, ok := spec.(*ast.TypeSpec)
						if !ok {
							continue
						}

						if ts.Name.Name == se.Sel.Name {
							sp := &StructParameters{
								ExternalPackage: se.Sel.Name,
								Imports:         fileImports,
							}
							if err := parseTypeSpec(syntax, sp, ts); err != nil {
								log.Fatal(err)
							}
							sp.Name = typeName
							if sp.Name != "" {
								structs.AddStruct(sp.Name, sp)
							}
						}
					}
				}
				// spew.Dump(syntax)
			}
		})
	}

	// map[fieldTypeName]*StructField
	exportedTypes := make(map[string]*StructField)
	// map[pkgName]map[pkgFieldTypeName]*StructField
	externalTypes := make(map[string]map[string]*StructField)
	for _, s := range structs {
		for _, field := range s.Fields {
			// exported fields
			if field.exprData.isExported {
				exportedTypes[field.Type] = field
			}

			// external fields
			if field.exprData.ixExternal {
				if _, ok := externalTypes[s.originalName]; !ok {
					externalTypes[field.exprData.pkgName] = make(map[string]*StructField)
				}
				externalTypes[field.exprData.pkgName][field.exprData.pkgType] = field
			}
		}
	}

	// find exported types in package
	if len(exportedTypes) > 0 {
		for _, pkg := range pkgs {
			for _, syntax := range pkg.Syntax {
				for _, decl := range syntax.Decls {
					genDecl, ok := decl.(*ast.GenDecl)
					if !ok {
						continue
					}

					for _, spec := range genDecl.Specs {
						ts, ok := spec.(*ast.TypeSpec)
						if !ok {
							continue
						}

						if _, ok := exportedTypes[ts.Name.Name]; !ok {
							continue
						}

						sp := &StructParameters{
							Imports: fileImports,
						}
						if err := parseTypeSpec(syntax, sp, ts); err != nil {
							log.Fatal(err)
						}
						sp.Name = ts.Name.Name
						if sp.Name != "" {
							structs.AddStruct(sp.Name, sp)
						}
					}
				}
			}
		}
	}

	// find extrnal
	if len(externalTypes) > 0 {
		packages.Visit(pkgs, nil, func(p *packages.Package) {
			for _, syntax := range p.Syntax {
				extPkg, ok := externalTypes[syntax.Name.Name]
				if !ok {
					continue
				}

				for _, decl := range syntax.Decls {
					genDecl, ok := decl.(*ast.GenDecl)
					if !ok {
						continue
					}

					for _, spec := range genDecl.Specs {
						ts, ok := spec.(*ast.TypeSpec)
						if !ok {
							continue
						}

						extType, ok := extPkg[ts.Name.Name]
						if !ok {
							continue
						}

						sp := &StructParameters{
							ExternalPackage: syntax.Name.Name,
							Imports:         fileImports,
						}
						if err := parseTypeSpec(syntax, sp, ts); err != nil {
							log.Fatal(err)
						}
						sp.Name = extType.Name
						if sp.Name != "" {
							structs.AddStruct(sp.Name, sp)
						}
					}
				}
			}
		})
	}

	return structs
}

func parseTypeSpec(node *ast.File, sp *StructParameters, spec *ast.TypeSpec) error {
	structType, ok := spec.Type.(*ast.StructType)
	if !ok {
		return nil
	}

	if spec.Name != nil {
		sp.Name = spec.Name.Name
		sp.originalName = spec.Name.Name
	}

	if structType.Fields != nil {
		for _, field := range structType.Fields.List {
			f := &StructField{}

			if field.Tag != nil {
				reTags := regexp.MustCompile(`(\w+):\"(\w+)\"`)
				match := reTags.FindAllStringSubmatch(field.Tag.Value, -1)
				f.Tags = make(map[string]string, len(match))
				for _, m := range match {
					f.Tags[m[1]] = m[2]
				}
			}

			fTypeData, err := parseTypeExpr(node, field.Type, true)
			if err != nil {
				log.Fatal(err)
			}

			if fTypeData.typeName == "" {
				continue
			}

			f.Type = fTypeData.typeName
			f.exprData = fTypeData

			for _, fName := range field.Names {
				f.Name = fName.Name
				sp.Fields = append(sp.Fields, f)
			}
		}
	}

	return nil
}

func GetStructsOld(file_models_str string) Structs {
	r := bufio.NewReader(strings.NewReader(file_models_str))

	structs := make(Structs)

	fileImports := utils.GetGoImportsFromFile(file_models_str)
	currentStruct := &StructParameters{
		Imports: fileImports,
	}

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}

		reCloseStruct := regexp.MustCompile(`^\}`)
		if reCloseStruct.MatchString(line) {
			structs[currentStruct.Name] = currentStruct
			currentStruct = &StructParameters{
				Imports: fileImports,
			}
			continue
		}

		reStartStruct := regexp.MustCompile(`type (\w+) struct {`)
		matches := reStartStruct.FindAllStringSubmatch(line, -1)
		if len(matches) == 1 {
			currentStruct = &StructParameters{
				Name:    matches[0][1],
				Fields:  make([]*StructField, 0),
				Imports: fileImports,
			}
			continue
		}

		// parse fields
		if currentStruct.Name != "" {
			reField := regexp.MustCompile(`\s*(?P<name>[\w\.]+)\s+(?P<type>[\w\*\.\[\]]+)\s+(?P<tags>\x60.+\x60)?`)
			match := reField.FindStringSubmatch(line)
			if len(match) == 0 {
				continue
			}

			field := StructField{Tags: make(map[string]string)}
			for index, name := range reField.SubexpNames() {
				if index != 0 && name != "" {
					switch name {
					case "name":
						field.Name = match[index]
					case "type":
						field.Type = match[index]
					case "tags":
						reTags := regexp.MustCompile(`(\w+):\"(\w+)\"`)
						match := reTags.FindAllStringSubmatch(match[index], -1)
						for _, m := range match {
							field.Tags[m[1]] = m[2]
						}
					}
				}
			}

			if field.Name != "" {
				currentStruct.Fields = append(currentStruct.Fields, &field)
			}
		}
	}

	return structs
}

func GetMissedStructs(s Structs, scalarTypes Types) []string {
	keys := make([]string, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}

	res := []string{}

	for _, st := range s {
		for _, stName := range st.GetNestedStructs(scalarTypes) {
			if !slices.Contains(keys, stName) && !slices.Contains(res, stName) {
				res = append(res, stName)
			}
		}
	}

	return res
}

func (s StructParameters) GetNestedStructs(scalarTypes Types) []string {
	res := []string{}

	for _, field := range s.Fields {
		tp := field.Type
		if tp[:1] == "*" {
			tp = tp[1:]
		}

		// skip lowercase type
		if tp != stringy.New(tp).UcFirst() {
			continue
		}

		if slices.Contains([]string{"[]byte"}, field.Type) {
			continue
		}

		_, existScalarType := scalarTypes[field.Type]
		if !slices.Contains(res, field.Type) && !existScalarType {
			res = append(res, field.Type)
		}
	}

	return res
}

func FillMissedTypes(allStructs Structs, modelsStructs Structs, scalarTypes Types) {
	missedStructs := GetMissedStructs(modelsStructs, scalarTypes)
	if len(missedStructs) == 0 {
		return
	}

	for _, st := range missedStructs {
		v, ok := allStructs[st]
		if !ok {
			_, ok := scalarTypes[st]
			if ok {
				continue
			}
			log.Fatalf("cannont find struct \"%s\"", st)
		}

		modelsStructs.AddStruct(st, v)
	}

	missedStructs = GetMissedStructs(modelsStructs, scalarTypes)
	if len(missedStructs) == 0 {
		return
	} else {
		FillMissedTypes(allStructs, modelsStructs, scalarTypes)
	}
}
