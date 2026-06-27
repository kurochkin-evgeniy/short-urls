// Package audit реализует паттерн «Наблюдатель» для аудита запросов.
package audit

import "time"

// Action описывает тип аудируемого запроса.
type Action string

const (
	// ActionShorten отправляется при создании короткой ссылки.
	ActionShorten Action = "shorten"
	// ActionFollow отправляется при переходе по короткой ссылке.
	ActionFollow Action = "follow"
)

// Event — JSON-сообщение, отправляемое приёмникам аудита.
type Event struct {
	URL    string `json:"url"`
	UserID string `json:"user_id,omitempty"`
	Action Action `json:"action"`
	TS     int64  `json:"ts"`
}

// Observer получает события аудита от Subject.
type Observer interface {
	Notify(event Event)
}

// Subject уведомляет зарегистрированных наблюдателей о событиях аудита.
type Subject struct {
	observers []Observer
}

// NewSubject создаёт субъект, пересылающий события указанным наблюдателям.
func NewSubject(observers ...Observer) *Subject {
	return &Subject{observers: observers}
}

// NewSubjectFromConfig создаёт субъект из настроек файла и удалённого URL аудита.
// Возвращает nil, если оба параметра пусты.
func NewSubjectFromConfig(auditFile, auditURL string) *Subject {
	var observers []Observer
	if auditFile != "" {
		observers = append(observers, NewFileObserver(auditFile))
	}
	if auditURL != "" {
		observers = append(observers, NewURLObserver(auditURL))
	}
	if len(observers) == 0 {
		return nil
	}
	return NewSubject(observers...)
}

// Notify формирует событие и отправляет его всем зарегистрированным наблюдателям.
func (s *Subject) Notify(action Action, userID, url string) {
	if s == nil {
		return
	}
	event := Event{
		TS:     time.Now().Unix(),
		Action: action,
		UserID: userID,
		URL:    url,
	}
	for _, o := range s.observers {
		o.Notify(event)
	}
}
