package structs

import (
	"fmt"
	"go/ast"
	"strings"
)

type fieldExprData struct {
	isExported bool
	ixExternal bool
	typeName   string
	pkgName    string
	pkgType    string
}

func parseTypeExpr(file *ast.File, typeExpr ast.Expr, ref bool) (*fieldExprData, error) {
	switch expr := typeExpr.(type) {
	// type Foo interface{}
	case *ast.InterfaceType:
		return nil, nil

	// type Foo struct {...}
	case *ast.StructType:
		return nil, fmt.Errorf("currently unavailable parse nested struct")

	// type Foo Baz
	case *ast.Ident:
		return &fieldExprData{
			isExported: expr.IsExported(),
			typeName:   expr.Name,
		}, nil

	// type Foo *Baz
	case *ast.StarExpr:
		t, err := parseTypeExpr(file, expr.X, ref)
		if err != nil {
			return nil, err
		}
		return &fieldExprData{
			isExported: t.isExported,
			typeName:   "*" + t.typeName,
		}, nil

	// type Foo pkg.Bar
	case *ast.SelectorExpr:
		if xIdent, ok := expr.X.(*ast.Ident); ok {
			fd := &fieldExprData{
				isExported: xIdent.IsExported(),
				ixExternal: true,
				typeName:   strings.Join([]string{xIdent.Name, expr.Sel.Name}, "."),
				pkgName:    xIdent.Name,
				pkgType:    expr.Sel.Name,
			}
			return fd, nil
		}

		return nil, fmt.Errorf("undefined selector expr")

	// type Foo []Baz
	case *ast.ArrayType:
		t, err := parseTypeExpr(file, expr.Elt, true)
		if err != nil {
			return nil, err
		}

		return &fieldExprData{
			isExported: t.isExported,
			typeName:   "[]" + t.typeName,
		}, nil

	// type Foo map[string]Bar
	case *ast.MapType:
		key, err := parseTypeExpr(file, expr.Value, true)
		if err != nil {
			return nil, err
		}

		if _, ok := expr.Value.(*ast.InterfaceType); ok {
			fd := &fieldExprData{
				typeName: fmt.Sprintf("map[%s]any", key.typeName),
			}
			return fd, nil
		}

		value, err := parseTypeExpr(file, expr.Value, true)
		if err != nil {
			return nil, err
		}

		fd := &fieldExprData{
			isExported: value.isExported,
			typeName:   fmt.Sprintf("map[%s]%s", key.typeName, value.typeName),
		}

		return fd, nil

	case *ast.FuncType:
		return nil, nil
	default:
		return nil, fmt.Errorf("unavailable type: %v", expr)
	}
}
