// functions are copied from golang.org/x/tools and modified to fit the current scenario.
// The original code is under the following license:
//
// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// Thanks to the Go Tool Authors for their great work!

package tool

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

type filedKind string

const (
	// From protocol.SymbolKind
	Field    filedKind = "field"
	Method   filedKind = "method"
	Variable filedKind = "variable"
	Constant filedKind = "constant"
)

func funcSymbol(decl *ast.FuncDecl) Symbol {
	s := Symbol{
		Kind:   "function",
		Name:   decl.Name.Name,
		Detail: types.ExprString(decl.Type),
		Doc:    "todo",
	}
	if decl.Recv != nil {
		s.Kind = "method"
	}

	// todo: receiver type name should be supported.
	// If function is a method, prepend the type of the method.
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		s.Name = fmt.Sprintf("(%s).%s", types.ExprString(decl.Recv.List[0].Type), s.Name)
	}

	return s
}

func typeSymbol(tf *token.File, spec *ast.TypeSpec) Symbol {
	s := Symbol{
		Kind: "type",
		Name: spec.Name.Name,
		Doc:  "todo",
	}

	s.Kind, s.Detail, s.Children = typeDetails(tf, spec.Type)
	return s
}

func typeDetails(tf *token.File, typExpr ast.Expr) (kind filedKind, detail string, children []DocumentSymbol) {
	switch typExpr := typExpr.(type) {
	case *ast.StructType:
		kind = "struct"
		children = fieldListSymbols(tf, typExpr.Fields, Field)
		if len(children) > 0 {
			detail = "struct{...}"
		} else {
			detail = "struct{}"
		}

		// Find interface methods and embedded types.
	case *ast.InterfaceType:
		kind = "interface"
		children = fieldListSymbols(tf, typExpr.Methods, Method)
		if len(children) > 0 {
			detail = "interface{...}"
		} else {
			detail = "interface{}"
		}

	case *ast.FuncType:
		kind = "function"
		detail = types.ExprString(typExpr)

	default:
		kind = "type"
		detail = types.ExprString(typExpr)
	}
	return
}

type DocumentSymbol struct {
	// The name of this symbol. Will be displayed in the user interface and therefore must not be
	// an empty string or a string only consisting of white spaces.
	Name string `json:"name"`
	// More detail for this symbol, e.g the signature of a function.
	Detail string `json:"detail,omitempty"`
	// The kind of this symbol.
	Kind filedKind `json:"kind"`
	// Indicates if this symbol is deprecated.
	//
	// @deprecated Use tags instead
	Deprecated bool `json:"deprecated,omitempty"`
	// Children of this symbol, e.g. properties of a class.
	// Children []DocumentSymbol `json:"children,omitempty"`
}

func fieldListSymbols(tf *token.File, fields *ast.FieldList, fieldKind filedKind) []DocumentSymbol {
	if fields == nil {
		return nil
	}

	var symbols []DocumentSymbol
	for _, field := range fields.List {
		detail, children := "", []DocumentSymbol(nil)
		if field.Type != nil {
			_, detail, children = typeDetails(tf, field.Type)
		}
		// todo: remove use of children because mcp calculates a cycle error.
		_ = children
		if len(field.Names) == 0 { // embedded interface or struct field
			// By default, use the formatted type details as the name of this field.
			// This handles potentially invalid syntax, as well as type embeddings in
			// interfaces.
			child := DocumentSymbol{
				Name: detail,
				Kind: Field, // consider all embeddings to be fields
				// Children: children,
				Detail: detail,
			}

			if id := embeddedIdent(field.Type); id != nil {
				child.Name = id.Name
				child.Detail = detail
			}
			symbols = append(symbols, child)
		} else {
			for _, name := range field.Names {
				child := DocumentSymbol{
					Name:   name.Name,
					Kind:   fieldKind,
					Detail: detail,
					// Children: children,
				}
				symbols = append(symbols, child)
			}
		}

	}
	return symbols
}

// embeddedIdent returns the type name identifier for an embedding x, if x in a
// valid embedding. Otherwise, it returns nil.
//
// Spec: An embedded field must be specified as a type name T or as a pointer
// to a non-interface type name *T
func embeddedIdent(x ast.Expr) *ast.Ident {
	if star, ok := x.(*ast.StarExpr); ok {
		x = star.X
	}
	switch ix := x.(type) { // check for instantiated receivers
	case *ast.IndexExpr:
		x = ix.X
	case *ast.IndexListExpr:
		x = ix.X
	}
	switch x := x.(type) {
	case *ast.Ident:
		return x
	case *ast.SelectorExpr:
		if _, ok := x.X.(*ast.Ident); ok {
			return x.Sel
		}
	}
	return nil
}

func varSymbol(tf *token.File, spec *ast.ValueSpec, name *ast.Ident, isConst bool) Symbol {
	s := Symbol{
		Name:   name.Name,
		Kind:   Variable,
		Doc:    "todo",
		Detail: types.ExprString(spec.Type),
	}
	if isConst {
		s.Kind = Constant
	}
	if spec.Type != nil { // type may be missing from the syntax
		_, s.Detail, s.Children = typeDetails(tf, spec.Type)
	}
	return s
}
