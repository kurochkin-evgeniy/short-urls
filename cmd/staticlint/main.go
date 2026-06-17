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
//   - exitcheck — пользовательский анализатор: запрещает прямой вызов os.Exit
//     в функции main пакета main; логику завершения следует выносить в run().
package main

import (
	"short-urls/cmd/staticlint/exitcheck"

	"github.com/timakin/bodyclose/passes/bodyclose"
	"golang.org/x/tools/go/analysis/multichecker"
	"honnef.co/go/tools/unused"
)

func main() {
	multichecker.Main(
		bodyclose.Analyzer,
		unused.Analyzer.Analyzer,
		exitcheck.Analyzer,
	)
}
