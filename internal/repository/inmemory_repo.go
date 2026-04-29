package repository

import (
	"encoding/json"
	"os"
	"short-urls/internal/logging"
	"sync"
)

type KeyValueStorage interface {
	InsertNewValue(key string, value string) (bool, string, error)
	InsertNewValuesBatch(items []BatchInsertItem) ([]BatchInsertResult, error)
	GetValue(key string) string
}

type BatchInsertItem struct {
	Key   string
	Value string
}
type BatchInsertResult struct {
	Inserted    bool
	ExistingKey string
}

type MapKeyValueStorage struct {
	mu            sync.RWMutex
	dict          map[string]string
	permanentFile string
}

type Filerecord struct {
	UUID        int    `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func NewMapKeyValueStorage() KeyValueStorage {

	return &MapKeyValueStorage{dict: make(map[string]string)}
}

func NewMapKeyValuePermanentStorage(path string) KeyValueStorage {
	st := &MapKeyValueStorage{dict: make(map[string]string), permanentFile: path}
	st.loadFromFile()
	return st
}

func (a *MapKeyValueStorage) InsertNewValue(key string, value string) (bool, string, error) {
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
	if err := a.saveToFile(); err != nil {
		return false, "", err
	}

	return true, "", nil
}

func (a *MapKeyValueStorage) InsertNewValuesBatch(items []BatchInsertItem) ([]BatchInsertResult, error) {
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

func (a *MapKeyValueStorage) GetValue(key string) string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.dict[key]
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
	}

}

func (a *MapKeyValueStorage) saveToFile() error {

	if a.permanentFile == "" {
		return nil
	}

	var records = make([]Filerecord, 0, len(a.dict))

	index := 0
	for key, value := range a.dict {
		records = append(records, Filerecord{UUID: index, ShortURL: key, OriginalURL: value})
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
