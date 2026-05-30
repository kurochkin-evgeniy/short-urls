package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingObserver struct {
	events []Event
}

func (r *recordingObserver) Notify(event Event) {
	r.events = append(r.events, event)
}

func TestSubjectNotify(t *testing.T) {
	rec := &recordingObserver{}
	subject := NewSubject(rec)

	subject.Notify(ActionShorten, "user-1", "https://example.com")

	require.Len(t, rec.events, 1)
	assert.Equal(t, ActionShorten, rec.events[0].Action)
	assert.Equal(t, "user-1", rec.events[0].UserID)
	assert.Equal(t, "https://example.com", rec.events[0].URL)
	assert.NotZero(t, rec.events[0].TS)
}

func TestSubjectNotifyNil(t *testing.T) {
	var subject *Subject
	subject.Notify(ActionFollow, "", "https://example.com")
}

func TestNewSubjectFromConfigEmpty(t *testing.T) {
	assert.Nil(t, NewSubjectFromConfig("", ""))
}

func TestNewSubjectFromConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")
	subject := NewSubjectFromConfig(path, "")
	require.NotNil(t, subject)

	subject.Notify(ActionShorten, "user-1", "https://example.com")

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var event Event
	require.NoError(t, json.Unmarshal(data[:len(data)-1], &event))
	assert.Equal(t, ActionShorten, event.Action)
}

func TestFileObserverNotify(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")
	observer := NewFileObserver(path)

	observer.Notify(Event{
		TS:     123,
		Action: ActionFollow,
		URL:    "https://example.com/path",
	})

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"action":"follow"`)
}
