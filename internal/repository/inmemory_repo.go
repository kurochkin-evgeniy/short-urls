package repository

import (
	"encoding/json"
	"os"
	"short-urls/internal/logging"
	"sync"
)

type KeyValueStorage interface {
	InsertNewValue(key string, value string, userID string) (bool, string, error)
	InsertNewValuesBatch(items []BatchInsertItem, userID string) ([]BatchInsertResult, error)
	LookupShortURL(key string) (originalURL string, isDeleted bool, found bool)
	GetUserURLs(userID string) ([]UserURL, error)
	MarkURLsDeletedBatch(userID string, shortURLs []string) error
}

type BatchInsertItem struct {
	Key   string
	Value string
}
type BatchInsertResult struct {
	Inserted    bool
	ExistingKey string
}

type UserURL struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type MapKeyValueStorage struct {
	mu            sync.RWMutex
	dict          map[string]string
	urlOwners     map[string]string
	deleted       map[string]bool
	permanentFile string
}

type Filerecord struct {
	UUID        int    `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	IsDeleted   bool   `json:"is_deleted"`
}

func NewMapKeyValueStorage() KeyValueStorage {

	return &MapKeyValueStorage{
		dict:      make(map[string]string),
		urlOwners: make(map[string]string),
		deleted:   make(map[string]bool),
	}
}

func NewMapKeyValuePermanentStorage(path string) KeyValueStorage {
	st := &MapKeyValueStorage{
		dict:          make(map[string]string),
		urlOwners:     make(map[string]string),
		deleted:       make(map[string]bool),
		permanentFile: path,
	}
	st.loadFromFile()
	return st
}

func (a *MapKeyValueStorage) InsertNewValue(key string, value string, userID string) (bool, string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for existingKey, existingValue := range a.dict {
		if existingValue == value {
			return false, existingKey, nil
		}
	}

	if _, ok := a.dict[key]; ok {
		return false, "", nil
	}
	a.dict[key] = value
	a.urlOwners[key] = userID
	delete(a.deleted, key)
	if err := a.saveToFile(); err != nil {
		return false, "", err
	}

	return true, "", nil
}

func (a *MapKeyValueStorage) InsertNewValuesBatch(items []BatchInsertItem, userID string) ([]BatchInsertResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	results := make([]BatchInsertResult, 0, len(items))
	for _, item := range items {
		inserted := false
		existingKey := ""
		for currentKey, currentValue := range a.dict {
			if currentValue == item.Value {
				existingKey = currentKey
				break
			}
		}
			if existingKey == "" {
				if _, exists := a.dict[item.Key]; !exists {
					a.dict[item.Key] = item.Value
					a.urlOwners[item.Key] = userID
					delete(a.deleted, item.Key)
					inserted = true
				}
			}
		results = append(results, BatchInsertResult{
			Inserted:    inserted,
			ExistingKey: existingKey,
		})
	}
	if err := a.saveToFile(); err != nil {
		return nil, err
	}
	return results, nil
}

func (a *MapKeyValueStorage) LookupShortURL(key string) (string, bool, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	v, ok := a.dict[key]
	if !ok {
		return "", false, false
	}
	return v, a.deleted[key], true
}

func (a *MapKeyValueStorage) MarkURLsDeletedBatch(userID string, shortURLs []string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, short := range shortURLs {
		if a.urlOwners[short] == userID {
			a.deleted[short] = true
		}
	}
	return a.saveToFile()
}

func (a *MapKeyValueStorage) GetUserURLs(userID string) ([]UserURL, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make([]UserURL, 0)
	for shortURL, ownerID := range a.urlOwners {
		if ownerID == userID && !a.deleted[shortURL] {
			result = append(result, UserURL{
				ShortURL:    shortURL,
				OriginalURL: a.dict[shortURL],
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
		a.dict[value.ShortURL] = value.OriginalURL
		if value.IsDeleted {
			a.deleted[value.ShortURL] = true
		}
	}

}

func (a *MapKeyValueStorage) saveToFile() error {

	if a.permanentFile == "" {
		return nil
	}

	var records = make([]Filerecord, 0, len(a.dict))

	index := 0
	for key, value := range a.dict {
		records = append(records, Filerecord{
			UUID:        index,
			ShortURL:    key,
			OriginalURL: value,
			IsDeleted:   a.deleted[key],
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
