// Usage:
//  Instrument: ./jeb <pkg|file.go>
//  Server: ./jeb
// Put /tmp first on your GOPATH, then run your app as normal
// (or if a binary run from /tmp/src/<pkg>)
package main

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
)

const (
	insertPkg      = "jeb/client"
	insertFuncPkg  = "client"
	insertFuncName = "Trace"
)

func main() {
	if len(os.Args) == 2 {
		prepare(os.Args[1])
	} else {
		runServer()
	}
}

func prepare(target string) {
	var fset *token.FileSet
	var f *ast.File
	if strings.HasSuffix(target, ".go") {
		fset, f = single(target)
	} else {
		fset, f = group(target)
	}
	instrument(fset, f)
	write(target, fset, f)
}

func single(filename string) (*token.FileSet, *ast.File) {
	sanity, err := os.Open(filename)
	if err != nil {
		log.Println(err)
		log.Fatal("Could not open ", filename, ". Is that path right?")
	}
	sanity.Close()

	fset := new(token.FileSet)
	f, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		log.Fatal("Error in ParseFile: ", err)
	}
	return fset, f
}

// group loads all the files in the package, and merges them into
// a single ast.File
func group(fullPkgName string) (*token.FileSet, *ast.File) {
	pkg, err := build.Import(fullPkgName, "", build.FindOnly)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Found package", fullPkgName, "in", pkg.Dir)

	fset := new(token.FileSet)
	pkgs, err := parser.ParseDir(fset, pkg.Dir, nil, 0)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(pkgs)

	pkgName := path.Base(fullPkgName)
	astPackage, ok := pkgs[pkgName]
	if ok {
		log.Printf("Instrumenting package '%s'\n", pkgName)
	} else {
		astPackage, ok = pkgs["main"]
		if !ok {
			log.Fatalf("Could not find package 'main' or '%s'. Cannot continue.", pkgName)
		}
		log.Println("Instrumenting package 'main'")
	}

	f := ast.MergePackageFiles(
		astPackage,
		ast.FilterFuncDuplicates&ast.FilterUnassociatedComments,
	)

	return fset, f
}

// Instrument this ast.File
func instrument(fset *token.FileSet, f *ast.File) {
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
		instrumentFunction(fset, fdecl)
	}
}

// Write out a single instrumented file per package, to temp dir
func write(fullPkgName string, fset *token.FileSet, f *ast.File) {
	out, err := os.Create(outName(fullPkgName))
	if err != nil {
		log.Fatal(err)
	}
	err = format.Node(out, fset, f)
	if err != nil {
		log.Println(err)
		log.Println("ERROR. Writing AST to", out.Name())
		err = ast.Fprint(out, fset, f, nil)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// outName is the name of the instrumented file
func outName(inName string) string {
	rootDir := os.TempDir()
	outDir := fmt.Sprintf("%s/src/%s/", rootDir, inName)
	os.MkdirAll(outDir, os.ModePerm)
	out := outDir + "jeb.go"
	log.Println("Writing to ", out)
	return out
}

func instrumentFunction(fset *token.FileSet, fdecl *ast.FuncDecl) {
	funcname := fdecl.Name.String()
	instrumentBlock(fset, fdecl.Body, funcname)
}

func instrumentBlock(fset *token.FileSet, block *ast.BlockStmt, funcname string) {
	var finalList []ast.Stmt
	local := make(map[string]bool)

	for _, expr := range block.List {

		log.Printf("%T, %v", expr, expr)

		position := fset.Position(expr.Pos())
		finalList = append(
			finalList,
			makeCall(
				insertFuncPkg,
				insertFuncName,
				strconv.Quote("LINE"),
				strconv.Quote(position.Filename),
				strconv.Quote(fmt.Sprintf("%d", position.Line)),
				strconv.Quote(funcname),
			),
		)

		/*
			for varName := range local {
				finalList = append(
					finalList,
					makeCall(
						insertFuncPkg,
						insertFuncName,
						strconv.Quote("VAR"),
						strconv.Quote(varName),
						varName,	// TODO: Emit AST for fmt.Sprintf("%s", varName)
					),
				)
			}
		*/

		isCall, callName := isCallExpr(expr)
		if isCall {
			finalList = append(
				finalList,
				makeCall(
					insertFuncPkg,
					insertFuncName,
					strconv.Quote("ENTER"),
					strconv.Quote(callName),
				),
			)
		}

		finalList = append(finalList, expr)

		if isCall {
			finalList = append(
				finalList,
				makeCall(
					insertFuncPkg,
					insertFuncName,
					strconv.Quote("EXIT"),
					strconv.Quote(callName),
				),
			)
		}

		switch texpr := expr.(type) {
		case *ast.IfStmt:
			instrumentBlock(fset, texpr.Body, funcname)
		case *ast.ForStmt:
			instrumentBlock(fset, texpr.Body, funcname)
		case *ast.SwitchStmt:
			instrumentBlock(fset, texpr.Body, funcname)
		case *ast.SelectStmt:
			instrumentBlock(fset, texpr.Body, funcname)
		case *ast.AssignStmt:
			for _, expr := range texpr.Lhs {
				if ident, ok := expr.(*ast.Ident); ok {
					log.Printf("Found variable '%s'\n", ident.Name)
					local[ident.Name] = true
				}
			}
		}

	}
	block.List = finalList
}

func isCallExpr(expr ast.Stmt) (bool, string) {
	exprstmt, ok := expr.(*ast.ExprStmt)
	if !ok {
		return false, ""
	}
	callexpr, ok := exprstmt.X.(*ast.CallExpr)
	if !ok {
		return false, ""
	}

	var name string
	switch v := callexpr.Fun.(type) {
	case *ast.SelectorExpr:
		name = v.Sel.Name
	case *ast.Ident:
		name = v.Name
	}
	return true, name
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
