
.PHONY: test
test:
	go test -v ./...

.PHONY: test-cover
test-cover:
	go test ./internal/... -cover

.PHONY: bench
bench:
	cd internal/service && go test -bench=. -benchmem -run=^Benchmark
	cd internal/repository && go test -bench=. -benchmem -run=^Benchmark
	cd internal/handler && go test -bench=. -benchmem -run=^Benchmark

.PHONY: build
build:
	go build ./cmd/shortener

.PHONY: build-loadgen
build-loadgen:
	go build ./cmd/loadgen

.PHONY: profile
profile: build build-loadgen
	@powershell -ExecutionPolicy Bypass -File scripts/profile.ps1