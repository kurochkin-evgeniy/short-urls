package audit

import (
	"encoding/json"
	"os"
	"sync"
)

var auditNewline = []byte{'\n'}

// FileObserver дописывает события аудита в локальный файл в формате JSON Lines.
type FileObserver struct {
	file *os.File
	mu   sync.Mutex
}

// NewFileObserver создаёт наблюдателя, записывающего события в path.
// Файл открывается один раз при создании и остаётся открытым на время жизни наблюдателя.
func NewFileObserver(path string) *FileObserver {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return &FileObserver{}
	}
	return &FileObserver{file: file}
}

// Notify дописывает событие в виде JSON-строки в настроенный файл.
func (f *FileObserver) Notify(event Event) {
	if f.file == nil {
		return
	}

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	_, _ = f.file.Write(data)
	_, _ = f.file.Write(auditNewline)
}

// Close закрывает файл аудита. Повторный вызов безопасен.
func (f *FileObserver) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.file == nil {
		return nil
	}
	err := f.file.Close()
	f.file = nil
	return err
}
