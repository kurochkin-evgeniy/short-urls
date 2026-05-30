package service

import (
	"fmt"
	"short-urls/internal/repository"
	"testing"
)

func BenchmarkCreateShortUrl(b *testing.B) {
	storage := repository.NewMapKeyValueStorage()
	s := NewShortUrlService("http://localhost:8080", storage)

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		url := fmt.Sprintf("https://example.com/bench/%d", i)
		_, err := s.CreateShortUrl(url, "bench-user")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLookupShortURL(b *testing.B) {
	storage := repository.NewMapKeyValueStorage()
	s := NewShortUrlService("http://localhost:8080", storage)

	ids := make([]string, 1000)
	for i := range ids {
		result, err := s.CreateShortUrl(fmt.Sprintf("https://example.com/lookup/%d", i), "bench-user")
		if err != nil {
			b.Fatal(err)
		}
		ids[i] = result.ShortURL[len("http://localhost:8080/"):]
	}

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		s.LookupShortURL(ids[i%len(ids)])
	}
}

func BenchmarkCreateBatchShortUrls(b *testing.B) {
	storage := repository.NewMapKeyValueStorage()
	s := NewShortUrlService("http://localhost:8080", storage)
	urls := make([]string, 10)
	for i := range urls {
		urls[i] = fmt.Sprintf("https://example.com/batch/%d", i)
	}

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		batch := make([]string, len(urls))
		for j, u := range urls {
			batch[j] = fmt.Sprintf("%s/%d", u, i)
		}
		_, err := s.CreateBatchShortUrls(batch, "bench-user")
		if err != nil {
			b.Fatal(err)
		}
	}
}
