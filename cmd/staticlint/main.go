// Пакет main реализует multichecker — инструмент статического анализа Go-кода.
//
// # Запуск
//
// Соберите бинарник и проверьте проект:
//
//	go build -o staticlint ./cmd/staticlint
//	./staticlint ./...
//
// Интеграция с go vet:
//
//	go build -o staticlint ./cmd/staticlint
//	go vet "-vettool=$(pwd)/staticlint" ./...
//
// Без сборки:
//
//	go run ./cmd/staticlint ./...
//
// # Анализаторы
//
// Multichecker объединяет проверки в одной команде
// (golang.org/x/tools/go/analysis/multichecker):
//
//   - bodyclose — проверяет, что тело HTTP-ответа (response.Body) закрывается
//     после чтения; незакрытое тело приводит к утечке соединений.
//
//   - unused (U1000) — находит неиспользуемые константы, переменные, функции
//     и типы в пределах анализируемых пакетов (staticcheck.io).
//
//   - staticcheck (SA*) — все анализаторы класса SA из honnef.co/go/tools/staticcheck:
//     поиск логических ошибок, утечек ресурсов, некорректного API, гонок и других
//     дефектов. Полный список: https://staticcheck.dev/docs/checks/#SA
//
//   - exitcheck — пользовательский анализатор: запрещает прямой вызов os.Exit
//     в функции main пакета main; логику завершения следует выносить в run().
package main

import (
	"short-urls/cmd/staticlint/exitcheck"

	"github.com/timakin/bodyclose/passes/bodyclose"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/unused"
)

func main() {
	multichecker.Main(analyzers()...)
}

func analyzers() []*analysis.Analyzer {
	checks := []*analysis.Analyzer{
		bodyclose.Analyzer,
		unused.Analyzer.Analyzer,
		exitcheck.Analyzer,
	}
	for _, a := range staticcheck.Analyzers {
		checks = append(checks, a.Analyzer)
	}
	return checks
}
