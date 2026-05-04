package service

import (
	"math/rand"
	"short-urls/internal/logging"
	"short-urls/internal/repository"
	"sync"
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

const (
	deleteBatchSize       = 100
	maxDeleteFanInWorkers = 8
)

// LookupShortURL returns the stored URL, whether it is soft-deleted, and whether the short id exists.
func (s *ShortUrlService) LookupShortURL(id string) (originalURL string, isDeleted bool, found bool) {
	return s.storage.LookupShortURL(id)
}

// QueueUserURLsDeletion accepts ownership-checked soft deletes; work continues asynchronously.
func (s *ShortUrlService) QueueUserURLsDeletion(userID string, shortIDs []string) {
	if len(shortIDs) == 0 {
		return
	}
	ids := append([]string(nil), shortIDs...)
	go s.runUserURLsDeletion(userID, ids)
}

func (s *ShortUrlService) runUserURLsDeletion(userID string, ids []string) {
	n := min(maxDeleteFanInWorkers, len(ids))
	if n < 1 {
		n = 1
	}
	ch := make(chan string, deleteBatchSize)
	var wg sync.WaitGroup
	chunkSize := (len(ids) + n - 1) / n
	start := 0
	for w := 0; w < n; w++ {
		end := start + chunkSize
		if end > len(ids) {
			end = len(ids)
		}
		if start >= end {
			break
		}
		part := ids[start:end]
		start = end
		wg.Add(1)
		go func(p []string) {
			defer wg.Done()
			for _, id := range p {
				ch <- id
			}
		}(part)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()

	var batch []string
	for id := range ch {
		batch = append(batch, id)
		if len(batch) >= deleteBatchSize {
			s.flushDeleteBatch(userID, batch)
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		s.flushDeleteBatch(userID, batch)
	}
}

func (s *ShortUrlService) flushDeleteBatch(userID string, batch []string) {
	if err := s.storage.MarkURLsDeletedBatch(userID, batch); err != nil {
		logging.Sugar.Errorw("mark urls deleted batch", "error", err, "batch_size", len(batch))
	}
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
