package repository

import (
	"fmt"
	"testing"
)

func BenchmarkInsertNewValue(b *testing.B) {
	storage := NewMapKeyValueStorage()

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		key := fmt.Sprintf("key%06d", i%10000)
		value := fmt.Sprintf("https://example.com/%d", i)
		_, _, err := storage.InsertNewValue(key, value, "user")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLookupShortURL(b *testing.B) {
	storage := NewMapKeyValueStorage()
	for i := range 1000 {
		key := fmt.Sprintf("id%04d", i)
		storage.InsertNewValue(key, fmt.Sprintf("https://example.com/%d", i), "user")
	}

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		storage.LookupShortURL(fmt.Sprintf("id%04d", i%1000))
	}
}

func BenchmarkGetUserURLs(b *testing.B) {
	storage := NewMapKeyValueStorage()
	for i := range 500 {
		storage.InsertNewValue(fmt.Sprintf("k%04d", i), fmt.Sprintf("https://example.com/%d", i), "bench-user")
	}

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		_, err := storage.GetUserURLs("bench-user")
		if err != nil {
			b.Fatal(err)
		}
	}
}
