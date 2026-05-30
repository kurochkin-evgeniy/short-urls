package service

import (
	"math/rand"
	"short-urls/internal/audit"
	"short-urls/internal/logging"
	"short-urls/internal/repository"
	"strings"
	"time"
)

type ShortUrlService struct {
	storage repository.KeyValueStorage
	baseUrl string
	audit   *audit.Subject

	deleteQueue chan deleteQueueJob
}

type deleteQueueJob struct {
	userID   string
	shortIDs []string
}

type CreateShortURLResult struct {
	ShortURL    string
	WasInserted bool
}

const defaultDeleteQueueBuf = 128

func NewShortUrlService(baseURL string, storage repository.KeyValueStorage, auditSubject ...*audit.Subject) *ShortUrlService {
	s := &ShortUrlService{
		storage:     storage,
		baseUrl:     baseURL,
		deleteQueue: make(chan deleteQueueJob, defaultDeleteQueueBuf),
	}
	if len(auditSubject) > 0 {
		s.audit = auditSubject[0]
	}
	go s.runDeleteQueueConsumer()
	return s
}

func (s *ShortUrlService) AuditShorten(originalURL, userID string) {
	s.audit.Notify(audit.ActionShorten, userID, originalURL)
}

func (s *ShortUrlService) AuditFollow(originalURL, userID string) {
	s.audit.Notify(audit.ActionFollow, userID, originalURL)
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
				ShortURL:    s.buildShortURL(id),
				WasInserted: true,
			}, nil
		}
		if existingID != "" {
			return CreateShortURLResult{
				ShortURL:    s.buildShortURL(existingID),
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
		results := make([]string, len(batchResults))
		for i, storageResult := range batchResults {
			shortID := items[i].Key
			if !storageResult.Inserted && storageResult.ExistingKey != "" {
				shortID = storageResult.ExistingKey
			}
			results[i] = s.buildShortURL(shortID)
		}
		return results, nil
	}
}

const (
	deleteBatchSize      = 100
	deleteFlushTickerDur = 500 * time.Millisecond
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
	go func() {
		s.deleteQueue <- deleteQueueJob{userID: userID, shortIDs: ids}
	}()
}

func (s *ShortUrlService) runDeleteQueueConsumer() {
	ticker := time.NewTicker(deleteFlushTickerDur)
	defer ticker.Stop()

	batch := make([]string, 0, deleteBatchSize)
	var batchUser string

	for {
		select {
		case job := <-s.deleteQueue:
			for _, id := range job.shortIDs {
				if id == "" {
					continue
				}
				if len(batch) > 0 && job.userID != batchUser {
					s.flushDeleteBatch(batchUser, batch)
					batch = batch[:0]
				}
				batchUser = job.userID
				batch = append(batch, id)
				for len(batch) >= deleteBatchSize {
					s.flushDeleteBatch(batchUser, batch[:deleteBatchSize])
					batch = append(batch[:0], batch[deleteBatchSize:]...)
				}
			}
			if len(batch) > 0 && len(s.deleteQueue) == 0 {
				s.flushDeleteBatch(batchUser, batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) == 0 || len(s.deleteQueue) == 0 {
				continue
			}
			s.flushDeleteBatch(batchUser, batch)
			batch = batch[:0]
		}
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

	result := make([]repository.UserURL, len(storageItems))
	for i, item := range storageItems {
		result[i] = repository.UserURL{
			ShortURL:    s.buildShortURL(item.ShortURL),
			OriginalURL: item.OriginalURL,
		}
	}

	return result, nil
}

func (s *ShortUrlService) buildShortURL(id string) string {
	var b strings.Builder
	b.Grow(len(s.baseUrl) + 1 + len(id))
	b.WriteString(s.baseUrl)
	b.WriteByte('/')
	b.WriteString(id)
	return b.String()
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const shortIDLen = 6

func randStringBytes(n int) string {
	var b [shortIDLen]byte
	for i := range b[:n] {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b[:n])
}
