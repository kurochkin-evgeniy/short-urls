// Package service реализует бизнес-логику сокращения URL.
package service

import (
	"math/rand"
	"short-urls/internal/audit"
	"short-urls/internal/logging"
	"short-urls/internal/repository"
	"strings"
	"time"
)

// ShortUrlService координирует хранилище, генерацию URL и уведомления аудита.
type ShortUrlService struct {
	deleteQueue chan deleteQueueJob
	deleteDone  chan struct{}
	audit       *audit.Subject
	storage     repository.KeyValueStorage
	baseUrl     string
}

type deleteQueueJob struct {
	userID   string
	shortIDs []string
}

// CreateShortURLResult содержит результат одной операции сокращения.
type CreateShortURLResult struct {
	ShortURL    string
	WasInserted bool
}

const defaultDeleteQueueBuf = 128

// NewShortUrlService создаёт сервис на основе хранилища и необязательного субъекта аудита.
func NewShortUrlService(baseURL string, storage repository.KeyValueStorage, auditSubject ...*audit.Subject) *ShortUrlService {
	s := &ShortUrlService{
		storage:     storage,
		baseUrl:     baseURL,
		deleteQueue: make(chan deleteQueueJob, defaultDeleteQueueBuf),
		deleteDone:  make(chan struct{}),
	}
	if len(auditSubject) > 0 {
		s.audit = auditSubject[0]
	}
	go s.runDeleteQueueConsumer()
	return s
}

// AuditShorten записывает успешное создание URL во все настроенные приёмники аудита.
func (s *ShortUrlService) AuditShorten(originalURL, userID string) {
	s.audit.Notify(audit.ActionShorten, userID, originalURL)
}

// AuditFollow записывает успешный переход по ссылке во все настроенные приёмники аудита.
func (s *ShortUrlService) AuditFollow(originalURL, userID string) {
	s.audit.Notify(audit.ActionFollow, userID, originalURL)
}

// CreateShortUrl генерирует короткую ссылку или возвращает существующую для данного URL.
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

// CreateBatchShortUrls генерирует короткие ссылки для пакета оригинальных URL.
func (s *ShortUrlService) CreateBatchShortUrls(urls []string, userID string) ([]string, error) {
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

const (
	deleteBatchSize      = 100
	deleteFlushTickerDur = 500 * time.Millisecond
)

// LookupShortURL возвращает сохранённый URL, признак мягкого удаления и факт существования идентификатора.
func (s *ShortUrlService) LookupShortURL(id string) (originalURL string, isDeleted bool, found bool) {
	return s.storage.LookupShortURL(id)
}

// QueueUserURLsDeletion принимает запрос на мягкое удаление; обработка продолжается асинхронно.
func (s *ShortUrlService) QueueUserURLsDeletion(userID string, shortIDs []string) {
	if len(shortIDs) == 0 {
		return
	}
	s.deleteQueue <- deleteQueueJob{
		userID:   userID,
		shortIDs: append([]string(nil), shortIDs...),
	}
}

// Shutdown завершает фоновую обработку удалений и сбрасывает несохранённые данные.
func (s *ShortUrlService) Shutdown() {
	close(s.deleteQueue)
	<-s.deleteDone
}

func (s *ShortUrlService) runDeleteQueueConsumer() {
	ticker := time.NewTicker(deleteFlushTickerDur)
	defer ticker.Stop()
	defer close(s.deleteDone)

	batch := make([]string, 0, deleteBatchSize)
	var batchUser string

	for {
		select {
		case job, ok := <-s.deleteQueue:
			if !ok {
				if len(batch) > 0 {
					s.flushDeleteBatch(batchUser, batch)
				}
				return
			}
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

// GetUserURLs возвращает все активные короткие ссылки пользователя.
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

// GetStats возвращает количество сокращённых URL и пользователей в сервисе.
func (s *ShortUrlService) GetStats() (repository.Stats, error) {
	return s.storage.GetStats()
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
