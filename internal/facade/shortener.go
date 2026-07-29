// Package facade содержит общую бизнес-логику для HTTP- и gRPC-обработчиков.
package facade

import (
	"context"
	"errors"

	"short-urls/internal/auth"
	"short-urls/internal/repository"
	"short-urls/internal/service"
)

var (
	// ErrUnauthorized возвращается, если авторизационные данные не содержат идентификатор пользователя.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrNotFound возвращается, если короткая ссылка не найдена.
	ErrNotFound = errors.New("not found")
	// ErrGone возвращается, если короткая ссылка была удалена.
	ErrGone = errors.New("gone")
)

// Shortener инкапсулирует операции сервиса сокращения URL.
type Shortener struct {
	svc *service.ShortUrlService
}

// New создаёт фасад поверх сервиса сокращения URL.
func New(svc *service.ShortUrlService) *Shortener {
	return &Shortener{svc: svc}
}

// ShortenURL создаёт короткую ссылку для оригинального URL.
func (f *Shortener) ShortenURL(ctx context.Context, originalURL string) (string, bool, error) {
	userID := auth.UserIDFromContext(ctx)
	createResult, err := f.svc.CreateShortUrl(originalURL, userID)
	if err != nil {
		return "", false, err
	}
	f.svc.AuditShorten(originalURL, userID)
	return createResult.ShortURL, createResult.WasInserted, nil
}

// ExpandURL возвращает оригинальный URL по идентификатору короткой ссылки.
func (f *Shortener) ExpandURL(ctx context.Context, id string) (string, error) {
	originalURL, isDeleted, found := f.svc.LookupShortURL(id)
	if !found {
		return "", ErrNotFound
	}
	if isDeleted {
		return "", ErrGone
	}
	f.svc.AuditFollow(originalURL, auth.UserIDFromContext(ctx))
	return originalURL, nil
}

// ListUserURLs возвращает все активные короткие ссылки пользователя.
func (f *Shortener) ListUserURLs(ctx context.Context) ([]repository.UserURL, error) {
	if auth.AuthWasPresent(ctx) && auth.AuthHasNoID(ctx) {
		return nil, ErrUnauthorized
	}
	return f.svc.GetUserURLs(auth.UserIDFromContext(ctx))
}

// CreateBatchShortUrls создаёт короткие ссылки для пакета оригинальных URL.
func (f *Shortener) CreateBatchShortUrls(ctx context.Context, urls []string) ([]string, error) {
	return f.svc.CreateBatchShortUrls(urls, auth.UserIDFromContext(ctx))
}

// QueueUserURLsDeletion ставит короткие ссылки пользователя в очередь на удаление.
func (f *Shortener) QueueUserURLsDeletion(ctx context.Context, shortIDs []string) error {
	if auth.AuthWasPresent(ctx) && auth.AuthHasNoID(ctx) {
		return ErrUnauthorized
	}
	f.svc.QueueUserURLsDeletion(auth.UserIDFromContext(ctx), shortIDs)
	return nil
}

// GetStats возвращает агрегированную статистику сервиса.
func (f *Shortener) GetStats(ctx context.Context) (repository.Stats, error) {
	return f.svc.GetStats()
}
