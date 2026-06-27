// Package exitcheck предоставляет анализатор, запрещающий прямой вызов os.Exit
// в функции main пакета main.
//
// Прямой вызов os.Exit в main затрудняет тестирование и корректное завершение
// (например, сброс буферов логгера). Рекомендуемый паттерн — вынести логику
// в функцию run() error и обработать ошибку в main через log.Fatal или return.
package exitcheck

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer проверяет отсутствие прямых вызовов os.Exit в функции main пакета main.
var Analyzer = &analysis.Analyzer{
	Name: "exitcheck",
	Doc:  "запрещает прямой вызов os.Exit в функции main пакета main",
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}

func run(pass *analysis.Pass) (any, error) {
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	ins := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	ins.Preorder([]ast.Node{(*ast.FuncDecl)(nil)}, func(node ast.Node) {
		fn := node.(*ast.FuncDecl)
		if fn.Name.Name != "main" || fn.Body == nil {
			return
		}

		if file := pass.Fset.File(node.Pos()); file != nil && strings.Contains(file.Name(), "go-build") {
			return
		}

		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || !isOsExit(pass, call) {
				return true
			}

			pass.Reportf(call.Pos(), "прямой вызов os.Exit запрещён в функции main пакета main")
			return true
		})
	})

	return nil, nil
}

func isOsExit(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	fn, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
	if !ok || fn.Pkg() == nil {
		return false
	}

	return fn.Pkg().Path() == "os" && fn.Name() == "Exit"
}
