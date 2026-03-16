
.PHONY: test
test:
	go test -v ./...

.PHONY: post
post:
	curl -X POST -H "Content-Type: text/plain" -d 'https://practicum.yandex.ru/' http://localhost:8080
