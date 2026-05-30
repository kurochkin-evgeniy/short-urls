package audit

import (
	"encoding/json"
	"os"
	"sync"
)

// FileObserver дописывает события аудита в локальный файл в формате JSON Lines.
type FileObserver struct {
	path string
	mu   sync.Mutex
}

// NewFileObserver создаёт наблюдателя, записывающего события в path.
func NewFileObserver(path string) *FileObserver {
	return &FileObserver{path: path}
}

// Notify дописывает событие в виде JSON-строки в настроенный файл.
func (f *FileObserver) Notify(event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	file, err := os.OpenFile(f.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer file.Close()

	_, _ = file.Write(append(data, '\n'))
}
