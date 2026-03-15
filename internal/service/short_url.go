package service

import (
	"math/rand"
	"short-urls/internal/repository"
)

type ShortUrlService struct {
	storage repository.KeyValueStorage
}

func NewShortUrlService() *ShortUrlService {
	return &ShortUrlService{storage: repository.NewMapKeyValueStorage()}
}

func (s *ShortUrlService) CreateShortUrl(url string) string {
	id := randStringBytesSafe(s.storage)
	s.storage.WriteValue(id, url)
	return id
}

func (s *ShortUrlService) ResolveShortUrl(id string) string {
	return s.storage.GetValue(id)
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func randStringBytesSafe(storage repository.KeyValueStorage) string {
	for {
		s := randStringBytes(6)
		if !storage.HasKey(s) {
			return s
		}
	}
}
