# go-musthave-shortener-tpl

Шаблон репозитория для трека «Сервис сокращения URL».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-shortener-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

### Сравнение профилей до и после оптимизации

```
go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof
```

```
File: shortener.exe
Build ID: C:\Users\khrya\short-urls\shortener.exe2026-06-13 16:44:49.0838748 +0300 MSK
Type: inuse_space
Time: 2026-05-30 19:47:07 MSK
Showing nodes accounting for -2135.55kB, 45.46% of 4697.65kB total
      flat  flat%   sum%        cum   cum%
 -596.16kB 12.69% 12.69%  -596.16kB 12.69%  short-urls/internal/service.NewShortUrlService
    -514kB 10.94% 23.63%     -514kB 10.94%  bufio.NewWriterSize (inline)
    -513kB 10.92% 34.55%     -513kB 10.92%  runtime.mallocgc
 -512.25kB 10.90% 45.46%  -512.25kB 10.90%  io.ReadAll
 -512.17kB 10.90% 56.36%  -512.17kB 10.90%  net/http.Header.Clone (inline)
  512.03kB 10.90% 45.46%   512.03kB 10.90%  internal/poll.(*FD).pin
         0     0% 45.46%  -512.17kB 10.90%  github.com/go-chi/chi/middleware.(*basicWriter).WriteHeader
         0     0% 45.46% -1024.42kB 21.81%  github.com/go-chi/chi/v5.(*Mux).ServeHTTP
         0     0% 45.46% -1024.42kB 21.81%  github.com/go-chi/chi/v5.(*Mux).routeHTTP
         0     0% 45.46% -1024.42kB 21.81%  github.com/go-chi/chi/v5/middleware.(*Compressor).Handler-fm.(*Compressor).Handler.func1
         0     0% 45.46%  -512.17kB 10.90%  github.com/go-chi/chi/v5/middleware.(*compressResponseWriter).WriteHeader
         0     0% 45.46%   512.03kB 10.90%  internal/poll.(*FD).Read
         0     0% 45.46%  -596.16kB 12.69%  main.main
         0     0% 45.46%   512.03kB 10.90%  net.(*conn).Read
         0     0% 45.46%   512.03kB 10.90%  net.(*netFD).Read
         0     0% 45.46% -1538.42kB 32.75%  net/http.(*conn).serve
         0     0% 45.46%   512.03kB 10.90%  net/http.(*connReader).backgroundRead
         0     0% 45.46%  -512.17kB 10.90%  net/http.(*response).WriteHeader
         0     0% 45.46% -1024.42kB 21.81%  net/http.HandlerFunc.ServeHTTP
         0     0% 45.46%     -514kB 10.94%  net/http.newBufioWriterSize
         0     0% 45.46% -1024.42kB 21.81%  net/http.serverHandler.ServeHTTP
         0     0% 45.46%     -513kB 10.92%  runtime.allocm
         0     0% 45.46%  -596.16kB 12.69%  runtime.main
         0     0% 45.46%     -513kB 10.92%  runtime.mcall
         0     0% 45.46%     -513kB 10.92%  runtime.newm
         0     0% 45.46%     -513kB 10.92%  runtime.newobject
         0     0% 45.46%     -513kB 10.92%  runtime.park_m
         0     0% 45.46%     -513kB 10.92%  runtime.resetspinning
         0     0% 45.46%     -513kB 10.92%  runtime.schedule
         0     0% 45.46%     -513kB 10.92%  runtime.startm
         0     0% 45.46%     -513kB 10.92%  runtime.wakep
         0     0% 45.46%  -596.16kB 12.69%  short-urls/internal/app.(*ShortUrlApp).Start
         0     0% 45.46% -1024.42kB 21.81%  short-urls/internal/app.(*ShortUrlApp).Start.AuthMiddleware.func2.1
         0     0% 45.46%  -512.25kB 10.90%  short-urls/internal/app.(*ShortUrlApp).Start.HandleCreateShortUrRequest.func3
         0     0% 45.46%  -512.17kB 10.90%  short-urls/internal/app.(*ShortUrlApp).Start.HandleRedirectRequest.func8
         0     0% 45.46%  -512.25kB 10.90%  short-urls/internal/handler.handleCreateShortUrl
         0     0% 45.46%  -512.17kB 10.90%  short-urls/internal/handler.handleRedirectUrl
         0     0% 45.46% -1024.42kB 21.81%  short-urls/internal/middleware.DecompressRequestMiddleware.func1
         0     0% 45.46% -1024.42kB 21.81%  short-urls/internal/middleware.LoggingMiddleware.func1
```
