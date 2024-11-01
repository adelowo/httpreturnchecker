package main

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name: "httpreturnchecker",
	Doc:  "checks for proper return statements after response writer operations in HTTP handlers",
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		funcDecl := n.(*ast.FuncDecl)
		if !isHTTPHandler(funcDecl) || funcDecl.Body == nil {
			return
		}

		writerName := funcDecl.Type.Params.List[0].Names[0].Name

		// Get last non-empty statement
		lastStmt := getLastNonEmptyStatement(funcDecl.Body.List)
		if lastStmt == nil {
			return
		}

		ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
			exprStmt, ok := node.(*ast.ExprStmt)
			if !ok {
				return true
			}

			if !isWriteToResponseWriter(exprStmt, writerName) {
				return true
			}

			// If this is the last non-empty statement in the function, it's fine
			if exprStmt == lastStmt {
				return true
			}

			// Find the block this write belongs to
			block := findParentBlock(funcDecl.Body, exprStmt)
			if block == nil {
				return true
			}

			// Find position of write in its block
			pos := -1
			for i, stmt := range block.List {
				if stmt == exprStmt {
					pos = i
					break
				}
			}

			if pos == -1 {
				return true
			}

			// Check next non-empty statement for return
			hasReturn := false
			for i := pos + 1; i < len(block.List); i++ {
				stmt := block.List[i]
				// Skip empty statements
				if isEmptyStatement(stmt) {
					continue
				}
				// Found next non-empty statement
				if _, isReturn := stmt.(*ast.ReturnStmt); isReturn {
					hasReturn = true
				}
				break
			}

			if !hasReturn {
				pass.Reportf(exprStmt.Pos(), "response write operation must be followed by return unless it's the last statement")
			}

			return true
		})
	})

	return nil, nil
}

func getLastNonEmptyStatement(stmts []ast.Stmt) ast.Stmt {
	for i := len(stmts) - 1; i >= 0; i-- {
		if !isEmptyStatement(stmts[i]) {
			return stmts[i]
		}
	}
	return nil
}

func isEmptyStatement(stmt ast.Stmt) bool {
	switch stmt.(type) {
	case *ast.EmptyStmt:
		return true
	// Comments are not statements in Go's AST
	// Empty declarations
	case *ast.DeclStmt:
		return true
	}
	return false
}

func findParentBlock(root ast.Node, target ast.Node) *ast.BlockStmt {
	var result *ast.BlockStmt
	ast.Inspect(root, func(n ast.Node) bool {
		if block, ok := n.(*ast.BlockStmt); ok {
			for _, stmt := range block.List {
				if stmt == target {
					result = block
					return false
				}
			}
		}
		return true
	})
	return result
}

func isWriteToResponseWriter(expr *ast.ExprStmt, writerName string) bool {
	switch x := expr.X.(type) {
	case *ast.CallExpr:
		switch fun := x.Fun.(type) {
		case *ast.SelectorExpr:
			// Direct w.Write, w.WriteHeader etc.
			if ident, ok := fun.X.(*ast.Ident); ok && ident.Name == writerName {
				return true
			}

			// Other write patterns
			if ident, ok := fun.X.(*ast.Ident); ok {
				switch ident.Name {
				case "json":
					return isJSONEncoderWrite(fun, x.Args, writerName)
				case "fmt":
					return isFmtWrite(fun, x.Args, writerName)
				case "io":
					return isIOCopy(fun, x.Args, writerName)
				// using github.com/go-chi/render
				case "render":
					return isRenderCall(fun, x.Args, writerName)
				}
			}
		}
	}
	return false
}

func isJSONEncoderWrite(sel *ast.SelectorExpr, args []ast.Expr, writerName string) bool {
	if sel.Sel.Name != "NewEncoder" || len(args) == 0 {
		return false
	}
	arg, ok := args[0].(*ast.Ident)
	return ok && arg.Name == writerName
}

func isFmtWrite(sel *ast.SelectorExpr, args []ast.Expr, writerName string) bool {
	if !isFmtPrint(sel.Sel.Name) || len(args) == 0 {
		return false
	}
	arg, ok := args[0].(*ast.Ident)
	return ok && arg.Name == writerName
}

func isIOCopy(sel *ast.SelectorExpr, args []ast.Expr, writerName string) bool {
	if sel.Sel.Name != "Copy" || len(args) == 0 {
		return false
	}
	arg, ok := args[0].(*ast.Ident)
	return ok && arg.Name == writerName
}

func isRenderCall(sel *ast.SelectorExpr, args []ast.Expr, writerName string) bool {
	if sel.Sel.Name != "Render" || len(args) == 0 {
		return false
	}
	arg, ok := args[0].(*ast.Ident)
	return ok && arg.Name == writerName
}

func isFmtPrint(name string) bool {
	return name == "Fprintf" || name == "Fprint" || name == "Fprintln"
}

func isHTTPHandler(funcDecl *ast.FuncDecl) bool {
	params := funcDecl.Type.Params
	if params == nil || len(params.List) != 2 {
		return false
	}
	return isResponseWriter(params.List[0].Type) && isHTTPRequest(params.List[1].Type)
}

func isResponseWriter(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if ident, ok := sel.X.(*ast.Ident); !ok || ident.Name != "http" {
		return false
	}
	return sel.Sel.Name == "ResponseWriter"
}

func isHTTPRequest(expr ast.Expr) bool {
	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if ident, ok := sel.X.(*ast.Ident); !ok || ident.Name != "http" {
		return false
	}
	return sel.Sel.Name == "Request"
}

