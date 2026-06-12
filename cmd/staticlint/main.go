// Package main содержит multichecker для статического анализа кода.
//
// Multichecker включает следующие анализаторы:
//
// Стандартные анализаторы golang.org/x/tools/go/analysis/passes:
//   - asmdecl - проверка соответствия ассемблерных вставок
//   - assign - проверка бесполезных присваиваний
//   - atomic - проверка использования sync/atomic
//   - bools - проверка подозрительных булевых выражений
//   - buildtag - проверка build-тегов
//   - cgocall - проверка вызовов cgo
//   - composite - проверка композитных литералов
//   - copylock - проверка копирования мьютексов
//   - deepequalerrors - проверка использования reflect.DeepEqual с ошибками
//   - defers - проверка использования defer
//   - directive - проверка директив компилятора
//   - errorsas - проверка использования errors.As
//   - fieldalignment - проверка выравнивания полей структур
//   - framepointer - проверка использования frame pointer
//   - httpresponse - проверка использования HTTP-ответов
//   - ifaceassert - проверка утверждений интерфейсов
//   - inspect - инспектирование AST
//   - loopclosure - проверка замыканий в циклах
//   - lostcancel - проверка потери контекста
//   - nilfunc - проверка вызовов nil-функций
//   - printf - проверка форматирования строк
//   - shift - проверка сдвигов
//   - sigchanyzer - проверка сигнальных каналов
//   - sortslice - проверка сортировки слайсов
//   - stdmethods - проверка сигнатур стандартных методов
//   - stringintconv - проверка конвертации строк в int
//   - structtag - проверка тегов структур
//   - tests - проверка тестов
//   - timeformat - проверка форматов времени
//   - unmarshal - проверка демаршалинга
//   - unreachable - проверка недостижимого кода
//   - unsafeptr - проверка использования unsafe.Pointer
//   - unusedresult - проверка неиспользуемых результатов
//
// Анализаторы SA пакета staticcheck.io:
//
//	Все анализаторы класса SA (static analysis):
//	SA1000-SA1030 - проверки корректности кода
//	SA4000-SA4030 - проверки использования типов
//	SA5000-SA5010 - проверки использования стандартной библиотеки
//	SA6000-SA6005 - проверки производительности
//	SA9000-SA9007 - проверки стиля кода
//
// Дополнительные анализаторы staticcheck.io (ST):
//
//	ST1000-ST1022 - проверки стиля кода
//
// Собственный анализатор:
//   - noexit - запрещает прямой вызов os.Exit в функции main пакета main
//
// Использование:
//
//	go build -o staticlint ./cmd/staticlint/
//	./staticlint ./...
//
// Или через go vet:
//
//	go vet -vettool=./staticlint ./...
//
// Для запуска с фильтрацией:
//
//	./staticlint -include=SA ./...
//	./staticlint -include=ST ./...
//	./staticlint -include=noexit ./...
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
	"golang.org/x/tools/go/analysis/passes/fieldalignment"
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

// NoExitAnalyzer - анализатор, запрещающий прямой вызов os.Exit в функции main пакета main
var NoExitAnalyzer = &analysis.Analyzer{
	Name: "noexit",
	Doc:  "запрещает прямой вызов os.Exit в функции main пакета main",
	Run:  runNoExit,
}

func runNoExit(pass *analysis.Pass) (interface{}, error) {
	// Проверяем, что пакет называется main
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		// Проверяем только файлы пакета main
		filename := pass.Fset.Position(file.Pos()).Filename
		if !strings.HasSuffix(filename, ".go") || strings.HasSuffix(filename, "_test.go") {
			continue
		}

		ast.Inspect(file, func(node ast.Node) bool {
			// Ищем функцию main
			funcDecl, ok := node.(*ast.FuncDecl)
			if !ok || funcDecl.Name.Name != "main" {
				return true
			}

			// Проверяем, что это функция без параметров
			if funcDecl.Type.Params != nil && len(funcDecl.Type.Params.List) > 0 {
				return true
			}

			// Ищем вызов os.Exit
			ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
				callExpr, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				// Проверяем, является ли вызов селектором (os.Exit)
				selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				// Проверяем, что вызывается именно os.Exit
				ident, ok := selExpr.X.(*ast.Ident)
				if !ok || ident.Name != "os" {
					return true
				}

				if selExpr.Sel.Name == "Exit" {
					pass.Reportf(callExpr.Pos(), "прямой вызов os.Exit в функции main пакета main запрещен")
				}

				return true
			})

			return false // Не нужно искать другие функции main
		})
	}

	return nil, nil
}

func main() {
	var analyzers []*analysis.Analyzer

	// Добавляем стандартные анализаторы
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
		fieldalignment.Analyzer,
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

	// Добавляем все анализаторы SA
	for _, a := range staticcheck.Analyzers {
		if strings.HasPrefix(a.Analyzer.Name, "SA") {
			analyzers = append(analyzers, a)
		}
	}

	// Добавляем анализаторы ST
	for _, a := range stylecheck.Analyzers {
		analyzers = append(analyzers, a)
	}

	// Добавляем собственный анализатор
	analyzers = append(analyzers, NoExitAnalyzer)

	multichecker.Main(analyzers...)
}
