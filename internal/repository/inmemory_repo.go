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
	sync.RWMutex
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

func (a *MapKeyValueStorage) InsertNewValue(key string, value string, userID string) (bool, string, error) {
	a.Lock()
	defer a.Unlock()

	if existingKey, ok := a.maps.originalToKey[value]; ok {
		return false, existingKey, nil
	}

	if _, ok := a.maps.dict[key]; ok {
		return false, "", nil
	}
	a.maps.dict[key] = value
	a.maps.urlOwners[key] = userID
	a.maps.originalToKey[value] = key
	a.maps.userKeys[userID] = append(a.maps.userKeys[userID], key)
	delete(a.maps.deleted, key)
	if err := a.saveToFile(); err != nil {
		return false, "", err
	}

	return true, "", nil
}

func (a *MapKeyValueStorage) InsertNewValuesBatch(items []BatchInsertItem, userID string) ([]BatchInsertResult, error) {
	a.Lock()
	defer a.Unlock()
	results := make([]BatchInsertResult, len(items))
	for i, item := range items {
		inserted := false
		existingKey := ""
		if key, ok := a.maps.originalToKey[item.Value]; ok {
			existingKey = key
		} else if _, exists := a.maps.dict[item.Key]; !exists {
			a.maps.dict[item.Key] = item.Value
			a.maps.urlOwners[item.Key] = userID
			a.maps.originalToKey[item.Value] = item.Key
			a.maps.userKeys[userID] = append(a.maps.userKeys[userID], item.Key)
			delete(a.maps.deleted, item.Key)
			inserted = true
		}
		results[i] = BatchInsertResult{
			ExistingKey: existingKey,
			Inserted:    inserted,
		}
	}
	if err := a.saveToFile(); err != nil {
		return nil, err
	}
	return results, nil
}

func (a *MapKeyValueStorage) LookupShortURL(key string) (string, bool, bool) {
	a.RLock()
	defer a.RUnlock()
	v, ok := a.maps.dict[key]
	if !ok {
		return "", false, false
	}
	return v, a.maps.deleted[key], true
}

func (a *MapKeyValueStorage) MarkURLsDeletedBatch(userID string, shortURLs []string) error {
	a.Lock()
	defer a.Unlock()
	for _, short := range shortURLs {
		if a.maps.urlOwners[short] == userID {
			a.maps.deleted[short] = true
		}
	}
	return a.saveToFile()
}

func (a *MapKeyValueStorage) GetUserURLs(userID string) ([]UserURL, error) {
	a.RLock()
	defer a.RUnlock()

	keys := a.maps.userKeys[userID]
	result := make([]UserURL, 0, len(keys))
	for _, shortURL := range keys {
		if a.maps.urlOwners[shortURL] == userID && !a.maps.deleted[shortURL] {
			result = append(result, UserURL{
				ShortURL:    shortURL,
				OriginalURL: a.maps.dict[shortURL],
			})
		}
	}

	return result, nil
}

func (a *MapKeyValueStorage) loadFromFile() {
	if a.permanentFile == "" {
		return
	}

	fileBytes, err := os.ReadFile(a.permanentFile)
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
		a.maps.dict[value.ShortURL] = value.OriginalURL
		a.maps.originalToKey[value.OriginalURL] = value.ShortURL
		if value.IsDeleted {
			a.maps.deleted[value.ShortURL] = true
		}
	}
}

func (a *MapKeyValueStorage) saveToFile() error {
	if a.permanentFile == "" {
		return nil
	}

	records := make([]Filerecord, 0, len(a.maps.dict))

	index := 0
	for key, value := range a.maps.dict {
		records = append(records, Filerecord{
			UUID:        index,
			ShortURL:    key,
			OriginalURL: value,
			IsDeleted:   a.maps.deleted[key],
		})
		index++
	}

	jsonData, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		logging.Sugar.Errorf("Failed MarshalIndent: Error = %s", err)
		return err
	}

	err = os.WriteFile(a.permanentFile, jsonData, 0644)
	if err != nil {
		logging.Sugar.Errorf("Failed WriteFile: Error = %s", err)
		return err
	}

	return nil
}
