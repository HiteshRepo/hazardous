package helpers

import (
	"fmt"
	"go/ast"
	"strings"

	"mvdan.cc/sh/syntax"
)

var assignmentOperators = map[string]any{
	":=": nil,
}

// TrackVariableAssignments traverses the given AST node and identifies variable assignments.
// It returns a map where the keys are variable names and the values are the corresponding assigned values.
// If a variable is not assigned a value, the value in the map will be an empty string.
//
// Parameters:
// - node: The AST node to traverse.
//
// Returns:
// - A map where the keys are variable names and the values are the corresponding assigned values.
func TrackVariableAssignments(node ast.Node) map[string]string {
	varMap := make(map[string]string)

	if node == nil {
		return varMap
	}

	ast.Inspect(node, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.AssignStmt:
			for _, lhs := range stmt.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok {
					if len(stmt.Rhs) > 0 {
						if rhsLit, ok := stmt.Rhs[0].(*ast.BasicLit); ok {
							varMap[ident.Name] = strings.Trim(rhsLit.Value, "\"")
						}
					}
				}
			}

		// case *ast.BlockStmt:
		// 	for _, stmt := range stmt.List {
		// 		return trackVariableAssignments(stmt)
		// 	}

		// case *ast.ExprStmt:
		// 	call, ok := stmt.X.(*ast.CallExpr)
		// 	if !ok {
		// 		return false
		// 	}

		// 	return trackVariableAssignments(call)

		case *ast.CallExpr:
			funIdent, ok := stmt.Fun.(*ast.Ident)
			if !ok {
				return false
			}

			for _, arg := range stmt.Args {
				ident, ok := arg.(*ast.Ident)
				if !ok {
					continue
				}

				if _, ok := assignmentOperators[ident.Name]; ok {
					// variables should be accompanied by assignment operators
					varMap[funIdent.Name] = ""
					continue
				}

				if _, ok := varMap[funIdent.Name]; ok {
					varMap[funIdent.Name] = ident.Name
					break
				}
			}
		}

		return true
	})

	return varMap
}

// ConvertSyntaxNodeToAstFile converts a given syntax.Node to an equivalent *ast.File.
// This function is designed to convert shell-like syntax to Go AST.
// It creates a new Go file named "main" and a function named "main" within it.
// The shell-like syntax is traversed and each command is converted to a corresponding Go call expression.
// The converted expressions are added to the "main" function's body.
//
// Parameters:
// - node: The syntax.Node to convert.
//
// Returns:
// - An equivalent *ast.File representing the converted shell-like syntax.
func ConvertSyntaxNodeToAstFile(node syntax.Node) *ast.File {
	f := &ast.File{
		Name:  ast.NewIdent("main"),
		Decls: []ast.Decl{},
	}

	funcDecl := &ast.FuncDecl{
		Name: ast.NewIdent("main"),
		Type: &ast.FuncType{
			Params:  &ast.FieldList{},
			Results: nil,
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{},
		},
	}

	syntax.Walk(node, func(n syntax.Node) bool {
		switch cmd := n.(type) {
		case *syntax.CallExpr:
			if len(cmd.Args) > 1 {
				varNameWord := cmd.Args[0]

				if len(varNameWord.Parts) == 1 {
					if varNameLit, ok := varNameWord.Parts[0].(*syntax.Lit); ok {
						goCall := &ast.CallExpr{
							Fun: ast.NewIdent(varNameLit.Value),
						}

						for _, arg := range cmd.Args[1:] {
							valueWord := arg
							valueWordStr := ""

							for _, vp := range valueWord.Parts {
								switch v := vp.(type) {
								case *syntax.Lit:
									valueWordStr = valueWordStr + v.Value

								case *syntax.DblQuoted:
									quotedVal := ""
									for _, part := range v.Parts {
										if lit, ok := part.(*syntax.Lit); ok {
											quotedVal += lit.Value
										}
									}

									valueWordStr = valueWordStr + quotedVal

								case *syntax.ParamExp:
									valueWordStr = valueWordStr + fmt.Sprintf("${%s}", v.Param.Value)
								}
							}

							goCall.Args = append(goCall.Args, ast.NewIdent(valueWordStr))
						}

						exprStmt := &ast.ExprStmt{X: goCall}
						funcDecl.Body.List = append(funcDecl.Body.List, exprStmt)
					}
				}
			}
		}

		return true
	})

	f.Decls = append(f.Decls, funcDecl)

	return f
}
