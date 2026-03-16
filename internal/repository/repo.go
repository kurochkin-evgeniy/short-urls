package repository

import "sync"

type KeyValueStorage interface {
	InsertNewValue(key string, value string) bool
	GetValue(key string) string
}

type MapKeyValueStorage struct {
	mu   sync.Mutex
	dict map[string]string
}

func NewMapKeyValueStorage() KeyValueStorage {
	return &MapKeyValueStorage{dict: make(map[string]string)}
}

func (a *MapKeyValueStorage) InsertNewValue(key string, value string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, ok := a.dict[key]; ok {
		return false
	}
	a.dict[key] = value

	return true
}

func (a *MapKeyValueStorage) GetValue(key string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.dict[key]
}
