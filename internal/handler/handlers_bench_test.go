package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"short-urls/internal/facade"
	"short-urls/internal/repository"
	"short-urls/internal/service"
)

func BenchmarkHandleCreateShortUrl(b *testing.B) {
	storage := repository.NewMapKeyValueStorage()
	f := facade.New(service.NewShortUrlService("http://localhost:8080", storage))
	h := http.HandlerFunc(HandleCreateShortUrRequest(f))

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		body := fmt.Sprintf("https://example.com/handler/%d", i)
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("Content-Type", "text/plain")
		rec := httptest.NewRecorder()
		h(rec, req)
		if rec.Code != http.StatusCreated {
			b.Fatalf("unexpected status: %d", rec.Code)
		}
	}
}

func BenchmarkHandleRedirectRequest(b *testing.B) {
	storage := repository.NewMapKeyValueStorage()
	s := service.NewShortUrlService("http://localhost:8080", storage)
	f := facade.New(s)
	result, err := s.CreateShortUrl("https://example.com/redirect", "user")
	if err != nil {
		b.Fatal(err)
	}
	shortID := strings.TrimPrefix(result.ShortURL, "http://localhost:8080/")
	h := http.HandlerFunc(HandleRedirectRequest(f))

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		req := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
		rec := httptest.NewRecorder()
		h(rec, req)
		if rec.Code != http.StatusTemporaryRedirect {
			b.Fatalf("unexpected status: %d", rec.Code)
		}
	}
}
