package audit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

// URLObserver отправляет события аудита на удалённый HTTP-эндпоинт методом POST.
type URLObserver struct {
	url    string
	client *http.Client
}

// NewURLObserver создаёт наблюдателя, отправляющего события POST-запросом на url.
func NewURLObserver(url string) *URLObserver {
	return &URLObserver{
		url: url,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Notify отправляет событие JSON POST-запросом на настроенный URL.
func (u *URLObserver) Notify(event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	req, err := http.NewRequest(http.MethodPost, u.url, bytes.NewReader(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := u.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}
