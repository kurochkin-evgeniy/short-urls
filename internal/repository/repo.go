package repository

import (
	"encoding/json"
	"os"
	"short-urls/internal/logging"
	"sync"
)

type KeyValueStorage interface {
	InsertNewValue(key string, value string) bool
	GetValue(key string) string
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

func (a *MapKeyValueStorage) InsertNewValue(key string, value string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, ok := a.dict[key]; ok {
		return false
	}
	a.dict[key] = value
	a.saveToFile()

	return true
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

func (a *MapKeyValueStorage) saveToFile() {

	if a.permanentFile == "" {
		return
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
		return
	}

	err = os.WriteFile(a.permanentFile, jsonData, 0644)
	if err != nil {
		logging.Sugar.Errorf("Failed WriteFile: Error = %s", err)
		return
	}

}
