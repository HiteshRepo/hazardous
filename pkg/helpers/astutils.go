package helpers

import (
	"fmt"
	"go/ast"
	"strings"

	"mvdan.cc/sh/syntax"
)

var assignmentOperators = map[string]any{
	":=": nil,
	"=":  nil,
	"+=": nil,
	"-=": nil,
	"*=": nil,
	"/=": nil,
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

	fileNode, ok := node.(*ast.File)
	if !ok || fileNode == nil {
		return varMap
	}

	ast.Inspect(fileNode, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.AssignStmt:
			if len(stmt.Lhs) == len(stmt.Rhs) {
				for i, lhs := range stmt.Lhs {
					lhsIdent, lhsOk := lhs.(*ast.Ident)
					if !lhsOk {
						continue
					}

					switch rhs := stmt.Rhs[i].(type) {
					case *ast.BasicLit:
						varMap[lhsIdent.Name] = strings.Trim(rhs.Value, "\"")

					case *ast.Ident:
						varMap[lhsIdent.Name] = rhs.Name

					case *ast.CallExpr:
						if funIdent, ok := rhs.Fun.(*ast.Ident); ok {
							var args []string
							for _, arg := range rhs.Args {
								switch arg := arg.(type) {
								case *ast.BasicLit:
									args = append(args, arg.Value)
								case *ast.Ident:
									args = append(args, arg.Name)
								default:
									args = append(args, "_")
								}
							}

							paramStr := "(" + strings.Join(args, ", ") + ")"
							varMap[lhsIdent.Name] = funIdent.Name + paramStr
						}
					}
				}
			}

		case *ast.ValueSpec:
			for i, name := range stmt.Names {
				if i < len(stmt.Values) {
					switch rhs := stmt.Values[i].(type) {
					case *ast.BasicLit:
						varMap[name.Name] = strings.Trim(rhs.Value, "\"")
					case *ast.Ident:
						varMap[name.Name] = rhs.Name
					case *ast.CallExpr:
						if funIdent, ok := rhs.Fun.(*ast.Ident); ok {
							var args []string
							for _, arg := range rhs.Args {
								switch arg := arg.(type) {
								case *ast.BasicLit:
									args = append(args, arg.Value)
								case *ast.Ident:
									args = append(args, arg.Name)

								default:
									args = append(args, "_")
								}
							}

							paramStr := "(" + strings.Join(args, ", ") + ")"
							varMap[name.Name] = funIdent.Name + paramStr
						}
					}
				} else {
					// Uninitialized variables
					varMap[name.Name] = ""
				}
			}

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

			return true
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

	// case *syntax.Assign:
	// 	for _, lhs := range stmt.Lhs {
	// 		if ident, ok := lhs.(*ast.Ident); ok {
	// 			if len(stmt.Rhs) > 0 {
	// 				if rhsLit, ok := stmt.Rhs[0].(*ast.BasicLit); ok {
	// 					varMap[ident.Name] = strings.Trim(rhsLit.Value, "\"")
	// 				}
	// 			}
	// 		}
	// 	}

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

			if len(cmd.Assigns) > 0 {
				varNameWord := cmd.Assigns[0]

				goCall := &ast.CallExpr{
					Fun: ast.NewIdent(varNameWord.Name.Value),
				}

				valueWord := varNameWord.Value
				valueWordStr := ""

				for _, vp := range valueWord.Parts {
					switch v := vp.(type) {
					case *syntax.Lit:
						valueWordStr = v.Value

					case *syntax.DblQuoted:
						quotedVal := ""
						for _, part := range v.Parts {
							if lit, ok := part.(*syntax.Lit); ok {
								quotedVal += lit.Value
							}
						}

						valueWordStr = quotedVal

					case *syntax.ParamExp:
						valueWordStr = fmt.Sprintf("${%s}", v.Param.Value)
					}

					goCall.Args = append(goCall.Args, ast.NewIdent(valueWordStr))
				}

				if len(goCall.Args) == 1 {
					last := goCall.Args[0]

					assignOp := ast.NewIdent("=")
					goCall.Args[0] = assignOp

					goCall.Args = append(goCall.Args, last)
				}

				exprStmt := &ast.ExprStmt{X: goCall}
				funcDecl.Body.List = append(funcDecl.Body.List, exprStmt)
			}
		}

		return true
	})

	f.Decls = append(f.Decls, funcDecl)

	return f
}
