package service

import (
	"math/rand"
	"short-urls/internal/repository"
)

type ShortUrlService struct {
	storage repository.KeyValueStorage
	baseUrl string
}

func NewShortUrlService(b string) *ShortUrlService {
	return &ShortUrlService{
		storage: repository.NewMapKeyValueStorage(),
		baseUrl: b,
	}
}

func (s *ShortUrlService) CreateShortUrl(url string) string {

	for {
		id := randStringBytes(6)
		if s.storage.InsertNewValue(id, url) {
			return s.baseUrl + "/" + id
		}
	}
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
