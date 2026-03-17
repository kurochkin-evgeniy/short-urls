package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
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

			s := service.NewShortUrlService("http://localhost:8080")

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

func Test_handleRedirectUrl400(t *testing.T) {

	s := service.NewShortUrlService("http://localhost:8080")

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h := http.HandlerFunc(HandleRedirectRequest(s))
	h(w, request)

	result := w.Result()

	assert.Equal(t, 400, result.StatusCode)
}

func Test_handleRedirectUrl307(t *testing.T) {

	const redirectUrl = "my url"
	s := service.NewShortUrlService("http://localhost:8080")
	url := s.CreateShortUrl(redirectUrl)

	request := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	h := http.HandlerFunc(HandleRedirectRequest(s))
	h(w, request)

	result := w.Result()

	assert.Equal(t, 307, result.StatusCode)
	assert.Equal(t, redirectUrl, w.Header().Get("Location"))
}

func Test_handleUnknownRedirectUrl400(t *testing.T) {

	const redirectUrl = "my url"
	s := service.NewShortUrlService("http://localhost:8080")

	request := httptest.NewRequest(http.MethodGet, "/1234", nil)
	w := httptest.NewRecorder()
	h := http.HandlerFunc(HandleRedirectRequest(s))
	h(w, request)

	result := w.Result()

	assert.Equal(t, 400, result.StatusCode)
}
