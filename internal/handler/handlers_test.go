package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"short-urls/internal/repository"
	"short-urls/internal/service"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_handleCreateShortUrl(t *testing.T) {

	type want struct {
		contentType string
		statusCode  int
	}

	tests := []struct {
		name        string
		request     string
		body        string
		contentType string
		want        want
	}{
		{
			name:        "simple test #1",
			request:     "/",
			body:        "123",
			contentType: "text/plain",
			want: want{
				contentType: "text/plain",
				statusCode:  201,
			},
		},
		{
			name:        "simple test #2",
			request:     "/test",
			body:        "123",
			contentType: "text/plain",
			want: want{
				contentType: "text/plain",
				statusCode:  201,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			s := service.NewShortUrlService("", repository.NewMapKeyValueStorage())

			request := httptest.NewRequest(http.MethodPost, tt.request, strings.NewReader(tt.body))
			request.Header.Add("Content-Type", "text/plain")
			w := httptest.NewRecorder()
			h := http.HandlerFunc(HandleCreateShortUrRequest(s))
			h(w, request)

			result := w.Result()

			assert.Equal(t, tt.want.statusCode, result.StatusCode)
			assert.Equal(t, tt.want.contentType, result.Header.Get("Content-Type"))

			bodyResult, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			err = result.Body.Close()
			require.NoError(t, err)

			assert.True(t, len(string(bodyResult)) != 0)
		})
	}
}

func Test_handleCreateShortUrlConflictTextPlain(t *testing.T) {
	s := service.NewShortUrlService("http://localhost:8080", repository.NewMapKeyValueStorage())
	body := "https://example.com/conflict"

	request1 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	request1.Header.Add("Content-Type", "text/plain")
	w1 := httptest.NewRecorder()
	http.HandlerFunc(HandleCreateShortUrRequest(s))(w1, request1)
	result1 := w1.Result()
	defer result1.Body.Close()
	require.Equal(t, http.StatusCreated, result1.StatusCode)
	firstBody, err := io.ReadAll(result1.Body)
	require.NoError(t, err)

	request2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	request2.Header.Add("Content-Type", "text/plain")
	w2 := httptest.NewRecorder()
	http.HandlerFunc(HandleCreateShortUrRequest(s))(w2, request2)
	result2 := w2.Result()
	defer result2.Body.Close()

	assert.Equal(t, http.StatusConflict, result2.StatusCode)
	assert.Equal(t, "text/plain", result2.Header.Get("Content-Type"))
	secondBody, err := io.ReadAll(result2.Body)
	require.NoError(t, err)
	assert.Equal(t, string(firstBody), string(secondBody))
}

func Test_handleCreateShortUrlConflictJSON(t *testing.T) {
	s := service.NewShortUrlService("http://localhost:8080", repository.NewMapKeyValueStorage())
	body := `{"url":"https://example.com/conflict-json"}`

	request1 := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	request1.Header.Add("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	http.HandlerFunc(HandleCreateShortUrRequest(s))(w1, request1)
	result1 := w1.Result()
	defer result1.Body.Close()
	require.Equal(t, http.StatusCreated, result1.StatusCode)
	firstBody, err := io.ReadAll(result1.Body)
	require.NoError(t, err)

	request2 := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	request2.Header.Add("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	http.HandlerFunc(HandleCreateShortUrRequest(s))(w2, request2)
	result2 := w2.Result()
	defer result2.Body.Close()

	assert.Equal(t, http.StatusConflict, result2.StatusCode)
	assert.Equal(t, "application/json", result2.Header.Get("Content-Type"))
	secondBody, err := io.ReadAll(result2.Body)
	require.NoError(t, err)
	assert.Equal(t, string(firstBody), string(secondBody))
}

func Test_handleRedirectUrl400(t *testing.T) {

	s := service.NewShortUrlService("", repository.NewMapKeyValueStorage())

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h := http.HandlerFunc(HandleRedirectRequest(s))
	h(w, request)

	result := w.Result()

	assert.Equal(t, 400, result.StatusCode)
}

func Test_handleCreateBatchShortUrl(t *testing.T) {
	s := service.NewShortUrlService("http://localhost:8080", repository.NewMapKeyValueStorage())

	body := `[
{"correlation_id":"id1","original_url":"https://example.com/1"},
{"correlation_id":"id2","original_url":"https://example.com/2"}
]`
	request := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(body))
	request.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h := http.HandlerFunc(HandleCreateBatchShortUrRequest(s))
	h(w, request)

	result := w.Result()
	defer result.Body.Close()

	assert.Equal(t, http.StatusCreated, result.StatusCode)
	assert.Equal(t, "application/json", result.Header.Get("Content-Type"))

	respBody, err := io.ReadAll(result.Body)
	require.NoError(t, err)

	var resp []BatchShortUrlResponse
	err = json.Unmarshal(respBody, &resp)
	require.NoError(t, err)
	require.Len(t, resp, 2)

	assert.Equal(t, "id1", resp[0].CorrelationID)
	assert.Equal(t, "id2", resp[1].CorrelationID)
	assert.True(t, strings.HasPrefix(resp[0].ShortURL, "http://localhost:8080/"))
	assert.True(t, strings.HasPrefix(resp[1].ShortURL, "http://localhost:8080/"))
}

func Test_handleCreateBatchShortUrlEmptyBatch(t *testing.T) {
	s := service.NewShortUrlService("http://localhost:8080", repository.NewMapKeyValueStorage())

	request := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader("[]"))
	request.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h := http.HandlerFunc(HandleCreateBatchShortUrRequest(s))
	h(w, request)

	result := w.Result()
	defer result.Body.Close()

	assert.Equal(t, http.StatusBadRequest, result.StatusCode)
}

func Test_handleRedirectUrl307(t *testing.T) {

	const redirectUrl = "my url"
	s := service.NewShortUrlService("", repository.NewMapKeyValueStorage())
	createResult := s.CreateShortUrl(redirectUrl)

	request := httptest.NewRequest(http.MethodGet, createResult.ShortURL, nil)
	w := httptest.NewRecorder()
	h := http.HandlerFunc(HandleRedirectRequest(s))
	h(w, request)

	result := w.Result()

	assert.Equal(t, 307, result.StatusCode)
	assert.Equal(t, redirectUrl, w.Header().Get("Location"))
}

func Test_handleUnknownRedirectUrl400(t *testing.T) {

	const redirectUrl = "my url"
	s := service.NewShortUrlService("", repository.NewMapKeyValueStorage())

	request := httptest.NewRequest(http.MethodGet, "/1234", nil)
	w := httptest.NewRecorder()
	h := http.HandlerFunc(HandleRedirectRequest(s))
	h(w, request)

	result := w.Result()

	assert.Equal(t, 400, result.StatusCode)
}
