// Package repository определяет интерфейсы и реализации хранилища коротких URL.
package repository

import (
	"encoding/json"
	"os"
	"short-urls/internal/logging"
	"sync"
)

// KeyValueStorage сохраняет и извлекает соответствия коротких URL.
type KeyValueStorage interface {
	InsertNewValue(key string, value string, userID string) (bool, string, error)
	InsertNewValuesBatch(items []BatchInsertItem, userID string) ([]BatchInsertResult, error)
	LookupShortURL(key string) (originalURL string, isDeleted bool, found bool)
	GetUserURLs(userID string) ([]UserURL, error)
	MarkURLsDeletedBatch(userID string, shortURLs []string) error
	GetStats() (Stats, error)
}

// Stats содержит агрегированную статистику сервиса.
type Stats struct {
	URLs  int
	Users int
}

// BatchInsertItem описывает одну запись в пакетной операции вставки.
type BatchInsertItem struct {
	Key   string
	Value string
}

// BatchInsertResult описывает результат вставки одного элемента пакета.
type BatchInsertResult struct {
	ExistingKey string
	Inserted    bool
}

// UserURL связывает короткий идентификатор с оригинальным URL в списке пользователя.
type UserURL struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type mapStorage struct {
	dict          map[string]string
	deleted       map[string]bool
	urlOwners     map[string]string
	originalToKey map[string]string
	userKeys      map[string][]string
}

// MapKeyValueStorage — in-memory реализация KeyValueStorage с опциональной записью в файл.
type MapKeyValueStorage struct {
	maps          mapStorage
	permanentFile string
	mu            sync.RWMutex
}

// Filerecord — JSON-представление сохранённого URL на диске.
type Filerecord struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UUID        int    `json:"uuid"`
	IsDeleted   bool   `json:"is_deleted"`
}

// NewMapKeyValueStorage возвращает пустое in-memory хранилище.
func NewMapKeyValueStorage() KeyValueStorage {
	return &MapKeyValueStorage{
		maps: mapStorage{
			dict:          make(map[string]string),
			urlOwners:     make(map[string]string),
			originalToKey: make(map[string]string),
			userKeys:      make(map[string][]string),
			deleted:       make(map[string]bool),
		},
	}
}

// NewMapKeyValuePermanentStorage загружает данные из path и сохраняет изменения обратно в файл.
func NewMapKeyValuePermanentStorage(path string) KeyValueStorage {
	st := &MapKeyValueStorage{
		maps: mapStorage{
			dict:          make(map[string]string),
			urlOwners:     make(map[string]string),
			originalToKey: make(map[string]string),
			userKeys:      make(map[string][]string),
			deleted:       make(map[string]bool),
		},
		permanentFile: path,
	}
	st.loadFromFile()
	return st
}

func (s *MapKeyValueStorage) InsertNewValue(key string, value string, userID string) (bool, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existingKey, ok := s.maps.originalToKey[value]; ok {
		return false, existingKey, nil
	}

	if _, ok := s.maps.dict[key]; ok {
		return false, "", nil
	}
	s.maps.dict[key] = value
	s.maps.urlOwners[key] = userID
	s.maps.originalToKey[value] = key
	s.maps.userKeys[userID] = append(s.maps.userKeys[userID], key)
	delete(s.maps.deleted, key)
	if err := s.saveToFile(); err != nil {
		return false, "", err
	}

	return true, "", nil
}

func (s *MapKeyValueStorage) InsertNewValuesBatch(items []BatchInsertItem, userID string) ([]BatchInsertResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	results := make([]BatchInsertResult, len(items))
	for i, item := range items {
		inserted := false
		existingKey := ""
		if key, ok := s.maps.originalToKey[item.Value]; ok {
			existingKey = key
		} else if _, exists := s.maps.dict[item.Key]; !exists {
			s.maps.dict[item.Key] = item.Value
			s.maps.urlOwners[item.Key] = userID
			s.maps.originalToKey[item.Value] = item.Key
			s.maps.userKeys[userID] = append(s.maps.userKeys[userID], item.Key)
			delete(s.maps.deleted, item.Key)
			inserted = true
		}
		results[i] = BatchInsertResult{
			ExistingKey: existingKey,
			Inserted:    inserted,
		}
	}
	if err := s.saveToFile(); err != nil {
		return nil, err
	}
	return results, nil
}

func (s *MapKeyValueStorage) LookupShortURL(key string) (string, bool, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.maps.dict[key]
	if !ok {
		return "", false, false
	}
	return v, s.maps.deleted[key], true
}

func (s *MapKeyValueStorage) MarkURLsDeletedBatch(userID string, shortURLs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, short := range shortURLs {
		if s.maps.urlOwners[short] == userID {
			s.maps.deleted[short] = true
		}
	}
	return s.saveToFile()
}

func (s *MapKeyValueStorage) GetUserURLs(userID string) ([]UserURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := s.maps.userKeys[userID]
	result := make([]UserURL, 0, len(keys))
	for _, shortURL := range keys {
		if s.maps.urlOwners[shortURL] == userID && !s.maps.deleted[shortURL] {
			result = append(result, UserURL{
				ShortURL:    shortURL,
				OriginalURL: s.maps.dict[shortURL],
			})
		}
	}

	return result, nil
}

func (s *MapKeyValueStorage) GetStats() (Stats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return Stats{
		URLs:  len(s.maps.dict),
		Users: len(s.maps.userKeys),
	}, nil
}

func (s *MapKeyValueStorage) loadFromFile() {
	if s.permanentFile == "" {
		return
	}

	fileBytes, err := os.ReadFile(s.permanentFile)
	if err != nil {
		logging.Sugar.Infof("Failed ReadFile: Error = %s", err)
		return
	}

	var result []Filerecord
	err = json.Unmarshal(fileBytes, &result)
	if err != nil {
		logging.Sugar.Infof("Failed Unmarshal: Error = %s", err)
		return
	}

	for _, value := range result {
		s.maps.dict[value.ShortURL] = value.OriginalURL
		s.maps.originalToKey[value.OriginalURL] = value.ShortURL
		if value.IsDeleted {
			s.maps.deleted[value.ShortURL] = true
		}
	}
}

func (s *MapKeyValueStorage) saveToFile() error {
	if s.permanentFile == "" {
		return nil
	}

	records := make([]Filerecord, 0, len(s.maps.dict))

	index := 0
	for key, value := range s.maps.dict {
		records = append(records, Filerecord{
			UUID:        index,
			ShortURL:    key,
			OriginalURL: value,
			IsDeleted:   s.maps.deleted[key],
		})
		index++
	}

	jsonData, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		logging.Sugar.Errorf("Failed MarshalIndent: Error = %s", err)
		return err
	}

	err = os.WriteFile(s.permanentFile, jsonData, 0644)
	if err != nil {
		logging.Sugar.Errorf("Failed WriteFile: Error = %s", err)
		return err
	}

	return nil
}

// Close сохраняет данные в файл, если хранилище использует постоянное хранение.
func (s *MapKeyValueStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveToFile()
}
