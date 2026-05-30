package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"short-urls/internal/logging"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddlewareSetsUserID(t *testing.T) {
	handler := AuthMiddleware("test-secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := UserIDFromContext(r.Context())
		assert.NotEmpty(t, userID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	cookies := rec.Result().Cookies()
	require.NotEmpty(t, cookies)
	assert.Equal(t, userCookieName, cookies[0].Name)
}

func TestUserIDFromContextEmpty(t *testing.T) {
	assert.Empty(t, UserIDFromContext(context.Background()))
}

func TestUserCookieFlags(t *testing.T) {
	ctx := context.WithValue(context.Background(), userCookieSeenContext, true)
	ctx = context.WithValue(ctx, userCookieNoIDContext, true)

	assert.True(t, UserCookieWasPresent(ctx))
	assert.True(t, UserCookieHasNoID(ctx))
}

func TestLoggingMiddleware(t *testing.T) {
	logging.LoggingInit()
	defer logging.LoggingDone()

	called := false
	handler := LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDecompressRequestMiddleware(t *testing.T) {
	handler := DecompressRequestMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
