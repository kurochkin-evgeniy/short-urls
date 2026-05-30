package audit

import "time"

type Action string

const (
	ActionShorten Action = "shorten"
	ActionFollow  Action = "follow"
)

type Event struct {
	TS     int64  `json:"ts"`
	Action Action `json:"action"`
	UserID string `json:"user_id,omitempty"`
	URL    string `json:"url"`
}

type Observer interface {
	Notify(event Event)
}

type Subject struct {
	observers []Observer
}

func NewSubject(observers ...Observer) *Subject {
	return &Subject{observers: observers}
}

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
