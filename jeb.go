package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strconv"
)

const (
	insertPkg      = "jeb/client"
	insertFuncPkg  = "client"
	insertFuncName = "Trace"
)

func main() {

	fset := new(token.FileSet)
	f, err := parser.ParseFile(fset, "example/simple.go", nil, 0)
	if err != nil {
		log.Fatal(err)
	}

	newImport := &ast.ImportSpec{
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: strconv.Quote(insertPkg),
		},
	}
	f.Imports = append(f.Imports, newImport)
	var isImportAdded bool

	// Instrument all the functions, and add the import
	for _, decl := range f.Decls {

		if !isImportAdded {
			gen, ok := decl.(*ast.GenDecl)
			if ok && gen.Tok == token.IMPORT {
				gen.Specs = append(gen.Specs, newImport)
				isImportAdded = true
			}
		}

		fdecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		log.Println("Instrumenting", fdecl.Name)
		instrument(fset, fdecl)
	}

	out, _ := os.Create("example/out.go")
	err = format.Node(out, fset, f)
	if err != nil {
		log.Fatal(err)
	}
}

func addImport(f *ast.File) {

}

func instrument(fset *token.FileSet, fdecl *ast.FuncDecl) {
	funcname := fdecl.Name.String()
	var finalList []ast.Stmt

	for _, expr := range fdecl.Body.List {
		position := fset.Position(expr.Pos())
		finalList = append(
			finalList,
			makeCall(
				insertFuncPkg,
				insertFuncName,
				strconv.Quote(position.Filename),
				strconv.Quote(fmt.Sprintf("%d", position.Line)),
				strconv.Quote(funcname),
			),
		)
		finalList = append(finalList, expr)
	}
	fdecl.Body.List = finalList
}

func makeCall(pkg, fname string, args ...string) *ast.ExprStmt {

	call := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent(pkg),
			Sel: ast.NewIdent(fname),
		},
		Args: []ast.Expr{},
	}
	for _, arg := range args {
		argsExpr := &ast.BasicLit{
			Kind:  token.STRING,
			Value: arg,
		}
		call.Args = append(call.Args, argsExpr)
	}

	return &ast.ExprStmt{X: call}
}

/*

 0: *ast.ExprStmt {
 .  X: *ast.CallExpr {
 .  .  Fun: *ast.SelectorExpr {
			X: *ast.Ident{Name: "fmt"}
			Sel: *ast.Ident{Name: "Println"}
 .  .  }
 .  .  Lparen: 4:9
 .  .  Args: []ast.Expr (len = 1) {
 .  .  .  0: *ast.BasicLit {
 .  .  .  .  ValuePos: 4:10
 .  .  .  .  Kind: STRING
 .  .  .  .  Value: "\"Hello, World!\""
 .  .  .  }
 .  .  }
 .  .  Ellipsis: -
 .  .  Rparen: 4:25
 .  }
 }

*/
