package service

import (
	"testing"
	"time"

	"short-urls/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateShortUrl(t *testing.T) {
	s := NewShortUrlService("http://localhost:8080", repository.NewMapKeyValueStorage())

	result, err := s.CreateShortUrl("https://example.com", "user-1")
	require.NoError(t, err)
	assert.True(t, result.WasInserted)
	assert.Contains(t, result.ShortURL, "http://localhost:8080/")
}

func TestCreateShortUrlDuplicate(t *testing.T) {
	storage := repository.NewMapKeyValueStorage()
	s := NewShortUrlService("http://localhost:8080", storage)

	first, err := s.CreateShortUrl("https://duplicate.com", "user-1")
	require.NoError(t, err)

	second, err := s.CreateShortUrl("https://duplicate.com", "user-2")
	require.NoError(t, err)
	assert.False(t, second.WasInserted)
	assert.Equal(t, first.ShortURL, second.ShortURL)
}

func TestLookupShortURL(t *testing.T) {
	s := NewShortUrlService("http://localhost:8080", repository.NewMapKeyValueStorage())
	created, err := s.CreateShortUrl("https://lookup.test", "user-1")
	require.NoError(t, err)

	shortID := created.ShortURL[len("http://localhost:8080/"):]
	url, deleted, found := s.LookupShortURL(shortID)
	assert.True(t, found)
	assert.False(t, deleted)
	assert.Equal(t, "https://lookup.test", url)
}

func TestCreateBatchShortUrls(t *testing.T) {
	s := NewShortUrlService("http://localhost:8080", repository.NewMapKeyValueStorage())
	urls := []string{
		"https://example.com/b1",
		"https://example.com/b2",
	}

	results, err := s.CreateBatchShortUrls(urls, "user-1")
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Contains(t, results[0], "http://localhost:8080/")
}

func TestGetUserURLs(t *testing.T) {
	s := NewShortUrlService("http://localhost:8080", repository.NewMapKeyValueStorage())
	_, err := s.CreateShortUrl("https://example.com/list", "user-1")
	require.NoError(t, err)

	urls, err := s.GetUserURLs("user-1")
	require.NoError(t, err)
	require.Len(t, urls, 1)
	assert.Equal(t, "https://example.com/list", urls[0].OriginalURL)
}

func TestQueueUserURLsDeletion(t *testing.T) {
	storage := repository.NewMapKeyValueStorage()
	s := NewShortUrlService("http://localhost:8080", storage)
	created, err := s.CreateShortUrl("https://example.com/delete", "user-1")
	require.NoError(t, err)

	shortID := created.ShortURL[len("http://localhost:8080/"):]
	s.QueueUserURLsDeletion("user-1", []string{shortID})

	time.Sleep(700 * time.Millisecond)

	_, deleted, found := s.LookupShortURL(shortID)
	require.True(t, found)
	assert.True(t, deleted)
}

func TestShutdownFlushesPendingDeletions(t *testing.T) {
	storage := repository.NewMapKeyValueStorage()
	s := NewShortUrlService("http://localhost:8080", storage)
	created, err := s.CreateShortUrl("https://example.com/shutdown", "user-1")
	require.NoError(t, err)

	shortID := created.ShortURL[len("http://localhost:8080/"):]
	s.QueueUserURLsDeletion("user-1", []string{shortID})
	s.Shutdown()

	_, deleted, found := s.LookupShortURL(shortID)
	require.True(t, found)
	assert.True(t, deleted)
}

func TestBuildShortURL(t *testing.T) {
	s := NewShortUrlService("http://localhost:8080", repository.NewMapKeyValueStorage())
	assert.Equal(t, "http://localhost:8080/abc123", s.buildShortURL("abc123"))
}
