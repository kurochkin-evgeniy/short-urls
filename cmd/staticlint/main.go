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
// ## golang.org/x/tools/go/analysis/passes
//
//   - appends — некорректное использование append;
//   - asmdecl — ошибки в объявлениях ассемблерных функций;
//   - assign — бесполезные присваивания;
//   - atomic — неверное использование sync/atomic;
//   - atomicalign — невыровненные 64-битные atomic-поля;
//   - bools — подозрительные логические выражения;
//   - buildtag — некорректные директивы go:build;
//   - cgocall — нарушения правил cgo;
//   - composite — некорректные литералы composite struct;
//   - copylock — копирование значений, содержащих sync.Locker;
//   - deepequalerrors — сравнение ошибок через == вместо errors.Is;
//   - defers — defer в циклах и другие антипаттерны;
//   - directive — некорректные go: директивы;
//   - errorsas — неверное использование errors.As;
//   - fieldalignment — неоптимальный порядок полей структуры;
//   - framepointer — несовместимость с framepointer;
//   - hostport — некорректные литералы host:port;
//   - httpmux — устаревший http.ServeMux;
//   - httpresponse — утечки HTTP response body;
//   - ifaceassert — небезопасные type assertion к интерфейсам;
//   - loopclosure — захват переменных цикла в замыканиях;
//   - lostcancel — context без вызова cancel;
//   - nilfunc — сравнение функции с nil;
//   - nilness — проверки на nil для заведомо ненулевых значений;
//   - printf — ошибки форматирования в Printf;
//   - reflectvaluecompare — сравнение reflect.Value через ==;
//   - scannererr — ошибки сканера go/scanner;
//   - shadow — затенение переменных;
//   - shift — бессмысленные побитовые сдвиги;
//   - sigchanyzer — буферизованный канал для os.Signal;
//   - slog — ошибки structured logging;
//   - sortslice — некорректные вызовы sort.Slice;
//   - sqlrowserr — необработанные ошибки sql.Rows;
//   - stdmethods — отсутствие стандартных методов (Error, String);
//   - stdversion — использование API из более новой версии Go;
//   - stringintconv — неверные преобразования string/int;
//   - structtag — некорректные struct tags;
//   - testinggoroutine — вызовы testing.T из горутин;
//   - tests — ошибки в тестах (TestXxx без t);
//   - timeformat — неверный порядок аргументов time.Format;
//   - unmarshal — ошибки json/xml unmarshal;
//   - unreachable — недостижимый код;
//   - unsafeptr — нарушения правил unsafe.Pointer;
//   - unusedresult — игнорирование обязательных результатов;
//   - unusedwrite — запись в неиспользуемые переменные;
//   - usesgenerics — некорректное использование дженериков;
//   - waitgroup — неверное использование sync.WaitGroup.
//
// ## staticcheck.io
//
//   - bodyclose — закрытие HTTP response.Body (сторонний анализатор);
//   - unused (U1000) — неиспользуемый код;
//   - staticcheck (SA*) — все проверки класса SA;
//   - stylecheck (ST1000) — комментарий к пакету должен начинаться с его имени.
//
// ## Пользовательский
//
//   - exitcheck — запрещает прямой вызов os.Exit в функции main пакета main.
package main

import (
	"short-urls/cmd/staticlint/exitcheck"

	"github.com/timakin/bodyclose/passes/bodyclose"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/appends"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/atomicalign"
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
	"golang.org/x/tools/go/analysis/passes/hostport"
	"golang.org/x/tools/go/analysis/passes/httpmux"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/reflectvaluecompare"
	"golang.org/x/tools/go/analysis/passes/scannererr"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/sqlrowserr"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stdversion"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/unusedwrite"
	"golang.org/x/tools/go/analysis/passes/usesgenerics"
	"golang.org/x/tools/go/analysis/passes/waitgroup"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
	"honnef.co/go/tools/unused"
)

func main() {
	multichecker.Main(analyzers()...)
}

func analyzers() []*analysis.Analyzer {
	checks := standardPasses()
	checks = append(checks,
		bodyclose.Analyzer,
		unused.Analyzer.Analyzer,
		exitcheck.Analyzer,
	)
	for _, a := range staticcheck.Analyzers {
		checks = append(checks, a.Analyzer)
	}
	checks = append(checks, stylecheck.Analyzers[0].Analyzer) // ST1000
	return checks
}

func standardPasses() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		appends.Analyzer,
		asmdecl.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		atomicalign.Analyzer,
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
		hostport.Analyzer,
		httpmux.Analyzer,
		httpresponse.Analyzer,
		ifaceassert.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		nilness.Analyzer,
		printf.Analyzer,
		reflectvaluecompare.Analyzer,
		scannererr.Analyzer,
		shadow.Analyzer,
		shift.Analyzer,
		sigchanyzer.Analyzer,
		slog.Analyzer,
		sortslice.Analyzer,
		sqlrowserr.Analyzer,
		stdmethods.Analyzer,
		stdversion.Analyzer,
		stringintconv.Analyzer,
		structtag.Analyzer,
		testinggoroutine.Analyzer,
		tests.Analyzer,
		timeformat.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,
		unusedwrite.Analyzer,
		usesgenerics.Analyzer,
		waitgroup.Analyzer,
	}
}
