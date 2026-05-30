package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapKeyValueStorageInsertAndLookup(t *testing.T) {
	storage := NewMapKeyValueStorage()

	inserted, existing, err := storage.InsertNewValue("abc123", "https://example.com", "user-1")
	require.NoError(t, err)
	assert.True(t, inserted)
	assert.Empty(t, existing)

	url, deleted, found := storage.LookupShortURL("abc123")
	assert.True(t, found)
	assert.False(t, deleted)
	assert.Equal(t, "https://example.com", url)
}

func TestMapKeyValueStorageDuplicateURL(t *testing.T) {
	storage := NewMapKeyValueStorage()

	_, _, err := storage.InsertNewValue("first1", "https://duplicate.com", "user-1")
	require.NoError(t, err)

	inserted, existing, err := storage.InsertNewValue("second", "https://duplicate.com", "user-2")
	require.NoError(t, err)
	assert.False(t, inserted)
	assert.Equal(t, "first1", existing)
}

func TestMapKeyValueStorageBatchInsert(t *testing.T) {
	storage := NewMapKeyValueStorage()
	items := []BatchInsertItem{
		{Key: "key001", Value: "https://example.com/1"},
		{Key: "key002", Value: "https://example.com/2"},
	}

	results, err := storage.InsertNewValuesBatch(items, "user-1")
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.True(t, results[0].Inserted)
	assert.True(t, results[1].Inserted)
}

func TestMapKeyValueStorageGetUserURLs(t *testing.T) {
	storage := NewMapKeyValueStorage()
	_, _, err := storage.InsertNewValue("url001", "https://example.com/u1", "user-1")
	require.NoError(t, err)
	_, _, err = storage.InsertNewValue("url002", "https://example.com/u2", "user-1")
	require.NoError(t, err)

	urls, err := storage.GetUserURLs("user-1")
	require.NoError(t, err)
	assert.Len(t, urls, 2)
}

func TestMapKeyValueStorageMarkDeleted(t *testing.T) {
	storage := NewMapKeyValueStorage()
	_, _, err := storage.InsertNewValue("del001", "https://example.com/del", "user-1")
	require.NoError(t, err)

	err = storage.MarkURLsDeletedBatch("user-1", []string{"del001"})
	require.NoError(t, err)

	_, deleted, found := storage.LookupShortURL("del001")
	assert.True(t, found)
	assert.True(t, deleted)
}

func TestMapKeyValueStorageLookupMissing(t *testing.T) {
	storage := NewMapKeyValueStorage()
	_, deleted, found := storage.LookupShortURL("missing")
	assert.False(t, found)
	assert.False(t, deleted)
}
