
.PHONY: test
test:
	go test -v ./...

.PHONY: post
post:
	curl -X POST -H "Content-Type: text/plain" -d 'https://practicum.yandex.ru/' http://localhost:8080


.PHONY: build
build:
	go build ./cmd/shortener