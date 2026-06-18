package main

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/defers"
	"golang.org/x/tools/go/analysis/passes/directive"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

var (
	ExitCheckAnalyzer = &analysis.Analyzer{
		Name: "exitcheck",
		Doc:  "запрещает вызовы panic, log.Fatal, log.Panic и os.Exit вне функции main.main",
		Run:  runExitCheck,
	}
)

func runExitCheck(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		filename := pass.Fset.Position(file.Pos()).Filename

		if strings.HasSuffix(filename, "_test.go") {
			continue
		}

		if strings.HasSuffix(filename, ".gen.go") {
			continue
		}

		ast.Inspect(file, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.FuncDecl:
				checkFunction(pass, n, file)
			}
			return true
		})
	}

	return nil, nil
}

func checkFunction(pass *analysis.Pass, fn *ast.FuncDecl, file *ast.File) {
	funcName := fn.Name.Name
	pkgName := file.Name.Name

	if strings.HasSuffix(funcName, "_test") || strings.HasPrefix(funcName, "Test") {
		return
	}

	isMainMain := pkgName == "main" && funcName == "main"

	if fn.Body != nil {
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			return checkNode(pass, n, funcName, pkgName, isMainMain)
		})
	}
}

func checkNode(pass *analysis.Pass, n ast.Node, funcName, pkgName string, isMainMain bool) bool {
	switch node := n.(type) {
	case *ast.CallExpr:
		return checkCallExpr(pass, node, funcName, pkgName, isMainMain)
	case *ast.GoStmt:
		ast.Inspect(node.Call, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				return checkCallExpr(pass, call, funcName+" (goroutine)", pkgName, false)
			}
			return true
		})
	case *ast.DeferStmt:
		ast.Inspect(node.Call, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				return checkCallExpr(pass, call, funcName+" (defer)", pkgName, false)
			}
			return true
		})
	}
	return true
}

func checkCallExpr(pass *analysis.Pass, call *ast.CallExpr, funcName, pkgName string, isMainMain bool) bool {
	if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "panic" {
		pass.Reportf(call.Pos(),
			"вызов panic в функции %s пакета %s запрещен (используйте возврат ошибок)",
			funcName, pkgName)
		return false
	}

	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			if ident.Name == "log" {
				if strings.HasPrefix(sel.Sel.Name, "Fatal") ||
					strings.HasPrefix(sel.Sel.Name, "Panic") {
					pass.Reportf(call.Pos(),
						"вызов log.%s в функции %s пакета %s запрещен (используйте возврат ошибок)",
						sel.Sel.Name, funcName, pkgName)
					return false
				}
			}

			if ident.Name == "os" && sel.Sel.Name == "Exit" {
				if !isMainMain {
					pass.Reportf(call.Pos(),
						"вызов os.Exit в функции %s пакета %s запрещен (разрешен только в main.main)",
						funcName, pkgName)
					return false
				}
			}

			if ident.Name == "logger" && (sel.Sel.Name == "Fatal" || sel.Sel.Name == "Panic") {
				pass.Reportf(call.Pos(),
					"вызов logger.%s в функции %s пакета %s запрещен (используйте возврат ошибок)",
					sel.Sel.Name, funcName, pkgName)
				return false
			}
		}
	}

	return true
}

func main() {
	var analyzers []*analysis.Analyzer

	analyzers = append(analyzers,
		asmdecl.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		bools.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		deepequalerrors.Analyzer,
		defers.Analyzer,
		directive.Analyzer,
		errorsas.Analyzer,
		framepointer.Analyzer,
		httpresponse.Analyzer,
		ifaceassert.Analyzer,
		inspect.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		printf.Analyzer,
		shift.Analyzer,
		sigchanyzer.Analyzer,
		sortslice.Analyzer,
		stdmethods.Analyzer,
		stringintconv.Analyzer,
		structtag.Analyzer,
		tests.Analyzer,
		timeformat.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,
	)

	for _, a := range staticcheck.Analyzers {
		if strings.HasPrefix(a.Analyzer.Name, "SA") {
			analyzers = append(analyzers, a.Analyzer)
		}
	}

	for _, a := range stylecheck.Analyzers {
		analyzers = append(analyzers, a.Analyzer)
	}

	analyzers = append(analyzers, ExitCheckAnalyzer)

	multichecker.Main(analyzers...)
}
