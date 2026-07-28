package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"short-urls/internal/auth"
	"short-urls/internal/logging"
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
	ctx := auth.ContextWithUserID(context.Background(), "", true, true)

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

func TestTrustedSubnetMiddleware(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("empty subnet forbids all", func(t *testing.T) {
		handler := TrustedSubnetMiddleware("")(okHandler)
		req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
		req.Header.Set("X-Real-IP", "192.168.1.10")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("ip in subnet allowed", func(t *testing.T) {
		handler := TrustedSubnetMiddleware("192.168.1.0/24")(okHandler)
		req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
		req.Header.Set("X-Real-IP", "192.168.1.10")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("ip outside subnet forbidden", func(t *testing.T) {
		handler := TrustedSubnetMiddleware("192.168.1.0/24")(okHandler)
		req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
		req.Header.Set("X-Real-IP", "10.0.0.1")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("missing header forbidden", func(t *testing.T) {
		handler := TrustedSubnetMiddleware("192.168.1.0/24")(okHandler)
		req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}
