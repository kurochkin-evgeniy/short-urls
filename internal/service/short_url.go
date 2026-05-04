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

func (s *ShortUrlService) CreateShortUrl(url string, userID string) (CreateShortURLResult, error) {

	for {
		id := randStringBytes(6)
		inserted, existingID, err := s.storage.InsertNewValue(id, url, userID)
		if err != nil {
			return CreateShortURLResult{}, err
		}
		if inserted {
			return CreateShortURLResult{
				ShortURL:    s.baseUrl + "/" + id,
				WasInserted: true,
			}, nil
		}
		if existingID != "" {
			return CreateShortURLResult{
				ShortURL:    s.baseUrl + "/" + existingID,
				WasInserted: false,
			}, nil
		}
	}
}

func (s *ShortUrlService) CreateBatchShortUrls(urls []string, userID string) ([]string, error) {

	for {
		items := make([]repository.BatchInsertItem, 0, len(urls))
		for _, url := range urls {
			items = append(items, repository.BatchInsertItem{
				Key:   randStringBytes(6),
				Value: url,
			})
		}
		batchResults, err := s.storage.InsertNewValuesBatch(items, userID)
		if err != nil {
			return nil, err
		}
		results := make([]string, 0, len(batchResults))
		for i, storageResult := range batchResults {
			shortID := items[i].Key
			if !storageResult.Inserted && storageResult.ExistingKey != "" {
				shortID = storageResult.ExistingKey
			}
			results = append(results,
				s.baseUrl+"/"+shortID,
			)
		}
		return results, nil
	}
}

func (s *ShortUrlService) ResolveShortUrl(id string) string {
	return s.storage.GetValue(id)
}

func (s *ShortUrlService) GetUserURLs(userID string) ([]repository.UserURL, error) {
	storageItems, err := s.storage.GetUserURLs(userID)
	if err != nil {
		return nil, err
	}

	result := make([]repository.UserURL, 0, len(storageItems))
	for _, item := range storageItems {
		result = append(result, repository.UserURL{
			ShortURL:    s.baseUrl + "/" + item.ShortURL,
			OriginalURL: item.OriginalURL,
		})
	}

	return result, nil
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
