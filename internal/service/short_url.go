package service

import (
	"math/rand"
	"short-urls/internal/repository"
)

type ShortUrlService struct {
	storage repository.KeyValueStorage
	baseUrl string
}

type CreateShortURLResult struct {
	ShortURL    string
	WasInserted bool
}

func NewShortUrlService(baseURL string, storage repository.KeyValueStorage) *ShortUrlService {
	return &ShortUrlService{
		storage: storage,
		baseUrl: baseURL,
	}
}

func (s *ShortUrlService) CreateShortUrl(url string) CreateShortURLResult {

	for {
		id := randStringBytes(6)
		inserted, existingID := s.storage.InsertNewValue(id, url)
		if inserted {
			return CreateShortURLResult{
				ShortURL:    s.baseUrl + "/" + id,
				WasInserted: true,
			}
		}
		if existingID != "" {
			return CreateShortURLResult{
				ShortURL:    s.baseUrl + "/" + existingID,
				WasInserted: false,
			}
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
