package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"short-urls/internal/facade"
	"short-urls/internal/middleware"
	"short-urls/internal/repository"
	"short-urls/internal/service"
	"strings"
	"testing"
	"time"

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

			f := facade.New(service.NewShortUrlService("", repository.NewMapKeyValueStorage()))

			request := httptest.NewRequest(http.MethodPost, tt.request, strings.NewReader(tt.body))
			request.Header.Add("Content-Type", "text/plain")
			w := httptest.NewRecorder()
			h := http.HandlerFunc(HandleCreateShortUrRequest(f))
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
	f := facade.New(s)
	body := "https://example.com/conflict"

	request1 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	request1.Header.Add("Content-Type", "text/plain")
	w1 := httptest.NewRecorder()
	http.HandlerFunc(HandleCreateShortUrRequest(f))(w1, request1)
	result1 := w1.Result()
	defer result1.Body.Close()
	require.Equal(t, http.StatusCreated, result1.StatusCode)
	firstBody, err := io.ReadAll(result1.Body)
	require.NoError(t, err)

	request2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	request2.Header.Add("Content-Type", "text/plain")
	w2 := httptest.NewRecorder()
	http.HandlerFunc(HandleCreateShortUrRequest(f))(w2, request2)
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
	f := facade.New(s)
	body := `{"url":"https://example.com/conflict-json"}`

	request1 := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	request1.Header.Add("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	http.HandlerFunc(HandleCreateShortUrRequest(f))(w1, request1)
	result1 := w1.Result()
	defer result1.Body.Close()
	require.Equal(t, http.StatusCreated, result1.StatusCode)
	firstBody, err := io.ReadAll(result1.Body)
	require.NoError(t, err)

	request2 := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	request2.Header.Add("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	http.HandlerFunc(HandleCreateShortUrRequest(f))(w2, request2)
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
	f := facade.New(s)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h := http.HandlerFunc(HandleRedirectRequest(f))
	h(w, request)

	result := w.Result()

	assert.Equal(t, 400, result.StatusCode)
}

func Test_handleCreateBatchShortUrl(t *testing.T) {
	s := service.NewShortUrlService("http://localhost:8080", repository.NewMapKeyValueStorage())
	f := facade.New(s)

	body := `[
{"correlation_id":"id1","original_url":"https://example.com/1"},
{"correlation_id":"id2","original_url":"https://example.com/2"}
]`
	request := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(body))
	request.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h := http.HandlerFunc(HandleCreateBatchShortUrRequest(f))
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
	f := facade.New(s)

	request := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader("[]"))
	request.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h := http.HandlerFunc(HandleCreateBatchShortUrRequest(f))
	h(w, request)

	result := w.Result()
	defer result.Body.Close()

	assert.Equal(t, http.StatusBadRequest, result.StatusCode)
}

func Test_handleRedirectUrl307(t *testing.T) {

	const redirectUrl = "my url"
	s := service.NewShortUrlService("", repository.NewMapKeyValueStorage())
	f := facade.New(s)
	createResult, err := s.CreateShortUrl(redirectUrl, "test-user")
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, createResult.ShortURL, nil)
	w := httptest.NewRecorder()
	h := http.HandlerFunc(HandleRedirectRequest(f))
	h(w, request)

	result := w.Result()

	assert.Equal(t, 307, result.StatusCode)
	assert.Equal(t, redirectUrl, w.Header().Get("Location"))
}

func Test_handleUnknownRedirectUrl400(t *testing.T) {

	const redirectUrl = "my url"
	s := service.NewShortUrlService("", repository.NewMapKeyValueStorage())
	f := facade.New(s)

	request := httptest.NewRequest(http.MethodGet, "/1234", nil)
	w := httptest.NewRecorder()
	h := http.HandlerFunc(HandleRedirectRequest(f))
	h(w, request)

	result := w.Result()

	assert.Equal(t, 400, result.StatusCode)
}

func Test_handleRedirectUrl410(t *testing.T) {
	st := repository.NewMapKeyValueStorage()
	s := service.NewShortUrlService("", st)
	f := facade.New(s)

	createResult, err := s.CreateShortUrl("https://example.com/gone", "")
	require.NoError(t, err)

	shortID := strings.TrimPrefix(createResult.ShortURL, "/")
	require.NotEmpty(t, shortID)

	require.NoError(t, st.MarkURLsDeletedBatch("", []string{shortID}))

	request := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
	w := httptest.NewRecorder()
	http.HandlerFunc(HandleRedirectRequest(f))(w, request)

	result := w.Result()
	defer result.Body.Close()
	assert.Equal(t, http.StatusGone, result.StatusCode)
}

func Test_handleGetUserURLs200(t *testing.T) {
	s := service.NewShortUrlService("http://localhost:8080", repository.NewMapKeyValueStorage())
	f := facade.New(s)
	auth := middleware.AuthMiddleware("test-secret")
	createHandler := auth(http.HandlerFunc(HandleCreateShortUrRequest(f)))
	getHandler := auth(http.HandlerFunc(HandleGetUserURLsRequest(f)))

	createReq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com/u1"))
	createReq.Header.Set("Content-Type", "text/plain")
	createRec := httptest.NewRecorder()
	createHandler.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusCreated, createRec.Result().StatusCode)

	cookies := createRec.Result().Cookies()
	require.NotEmpty(t, cookies)
	userCookie := cookies[0]

	getReq := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	getReq.AddCookie(userCookie)
	getRec := httptest.NewRecorder()
	getHandler.ServeHTTP(getRec, getReq)

	res := getRec.Result()
	defer res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Equal(t, "application/json", res.Header.Get("Content-Type"))

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	var payload []UserURLResponse
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Len(t, payload, 1)
	assert.Equal(t, "https://example.com/u1", payload[0].OriginalURL)
	assert.True(t, strings.HasPrefix(payload[0].ShortURL, "http://localhost:8080/"))
}

func Test_handleGetUserURLs204(t *testing.T) {
	s := service.NewShortUrlService("http://localhost:8080", repository.NewMapKeyValueStorage())
	f := facade.New(s)
	auth := middleware.AuthMiddleware("test-secret")
	getHandler := auth(http.HandlerFunc(HandleGetUserURLsRequest(f)))

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	rec := httptest.NewRecorder()
	getHandler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusNoContent, res.StatusCode)
}

func Test_handleGetUserURLs401WhenCookieHasNoUserID(t *testing.T) {
	s := service.NewShortUrlService("http://localhost:8080", repository.NewMapKeyValueStorage())
	f := facade.New(s)
	auth := middleware.AuthMiddleware("test-secret")
	getHandler := auth(http.HandlerFunc(HandleGetUserURLsRequest(f)))

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	req.AddCookie(&http.Cookie{
		Name:  "user_token",
		Value: makeSignedTokenWithEmptyUserID("test-secret"),
		Path:  "/",
	})
	rec := httptest.NewRecorder()
	getHandler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func makeSignedTokenWithEmptyUserID(secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(""))
	signature := mac.Sum(nil)
	raw := "." + hex.EncodeToString(signature)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func Test_handleDeleteUserURLs202(t *testing.T) {
	s := service.NewShortUrlService("http://localhost:8080", repository.NewMapKeyValueStorage())
	f := facade.New(s)
	auth := middleware.AuthMiddleware("test-secret")
	createHandler := auth(http.HandlerFunc(HandleCreateShortUrRequest(f)))
	deleteHandler := auth(http.HandlerFunc(HandleDeleteUserURLsRequest(f)))
	redirectHandler := http.HandlerFunc(HandleRedirectRequest(f))

	createReq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com/del"))
	createReq.Header.Set("Content-Type", "text/plain")
	createRec := httptest.NewRecorder()
	createHandler.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusCreated, createRec.Result().StatusCode)
	cookies := createRec.Result().Cookies()
	require.NotEmpty(t, cookies)
	userCookie := cookies[0]

	createBody, err := io.ReadAll(createRec.Result().Body)
	require.NoError(t, err)
	_ = createRec.Result().Body.Close()
	shortPath := strings.TrimPrefix(strings.TrimSpace(string(createBody)), "http://localhost:8080/")
	shortID := strings.TrimPrefix(shortPath, "/")

	delBody := `["` + shortID + `"]`
	delReq := httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader(delBody))
	delReq.Header.Set("Content-Type", "application/json")
	delReq.AddCookie(userCookie)
	delRec := httptest.NewRecorder()
	deleteHandler.ServeHTTP(delRec, delReq)
	require.Equal(t, http.StatusAccepted, delRec.Result().StatusCode)

	time.Sleep(150 * time.Millisecond)

	getReq := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
	getRec := httptest.NewRecorder()
	redirectHandler.ServeHTTP(getRec, getReq)
	res := getRec.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusGone, res.StatusCode)
}

func Test_handleDeleteUserURLs400EmptyBody(t *testing.T) {
	s := service.NewShortUrlService("http://localhost:8080", repository.NewMapKeyValueStorage())
	f := facade.New(s)
	auth := middleware.AuthMiddleware("test-secret")
	deleteHandler := auth(http.HandlerFunc(HandleDeleteUserURLsRequest(f)))

	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader("[]"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	deleteHandler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Result().StatusCode)
}

func Test_handleDeleteUserURLs401WhenCookieHasNoUserID(t *testing.T) {
	s := service.NewShortUrlService("http://localhost:8080", repository.NewMapKeyValueStorage())
	f := facade.New(s)
	auth := middleware.AuthMiddleware("test-secret")
	deleteHandler := auth(http.HandlerFunc(HandleDeleteUserURLsRequest(f)))

	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader(`["abc"]`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{
		Name:  "user_token",
		Value: makeSignedTokenWithEmptyUserID("test-secret"),
		Path:  "/",
	})
	rec := httptest.NewRecorder()
	deleteHandler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Result().StatusCode)
}

func Test_handleGetStats(t *testing.T) {
	s := service.NewShortUrlService("http://localhost:8080", repository.NewMapKeyValueStorage())
	f := facade.New(s)
	_, err := s.CreateShortUrl("https://example.com/1", "user-1")
	require.NoError(t, err)
	_, err = s.CreateShortUrl("https://example.com/2", "user-2")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
	rec := httptest.NewRecorder()
	http.HandlerFunc(HandleGetStatsRequest(f))(rec, req)

	result := rec.Result()
	defer result.Body.Close()
	require.Equal(t, http.StatusOK, result.StatusCode)
	assert.Equal(t, "application/json", result.Header.Get("Content-Type"))

	var resp StatsResponse
	require.NoError(t, json.NewDecoder(result.Body).Decode(&resp))
	assert.Equal(t, 2, resp.URLs)
	assert.Equal(t, 2, resp.Users)
}
