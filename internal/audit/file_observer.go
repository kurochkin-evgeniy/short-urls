package audit

import (
	"encoding/json"
	"os"
	"sync"
)

type FileObserver struct {
	path string
	mu   sync.Mutex
}

func NewFileObserver(path string) *FileObserver {
	return &FileObserver{path: path}
}

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
