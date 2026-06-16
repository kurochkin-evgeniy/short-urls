// Package middleware предоставляет HTTP-middleware для логирования, аутентификации и распаковки.
package middleware

import (
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"short-urls/internal/logging"
	"strings"
	"time"

	"github.com/go-chi/chi/middleware"
)

type contextKey string

const (
	userIDContextKey      contextKey = "userID"
	userCookieSeenContext contextKey = "userCookieSeen"
	userCookieNoIDContext contextKey = "userCookieNoID"
	userCookieName                   = "user_token"
	defaultCookieSecret              = "short-urls-secret"
)

// LoggingMiddleware логирует метод, путь, код ответа и длительность запроса.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		start := time.Now()
		next.ServeHTTP(ww, r)
		duration := time.Since(start)

		logging.Sugar.Infof("Request: %s %s duration=%v", r.Method, r.URL.Path, duration)
		logging.Sugar.Infof("Responce: %v size=%v", ww.Status(), ww.BytesWritten())
	})
}

// DecompressRequestMiddleware прозрачно распаковывает тело запроса, сжатое gzip.
func DecompressRequestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") == "gzip" {
			// Оборачиваем тело запроса gzip-ридером.
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}
			defer gz.Close()

			// Заменяем исходное тело запроса распакованным.
			r.Body = gz
			// Удаляем заголовки Content-Encoding и Content-Length.
			r.Header.Del("Content-Encoding")
			r.Header.Del("Content-Length")
		}
		// Передаём управление следующему обработчику.
		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware назначает идентификатор пользователя через подписанную cookie и сохраняет его в контексте запроса.
func AuthMiddleware(secret string) func(http.Handler) http.Handler {
	secretToUse := secret
	if secretToUse == "" {
		secretToUse = defaultCookieSecret
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(userCookieName)
			cookieWasPresent := err == nil

			userID := ""
			cookieWithNoID := false
			if cookieWasPresent {
				verifiedID, ok, hasID := verifySignedUserToken(cookie.Value, secretToUse)
				if ok {
					userID = verifiedID
					if !hasID {
						cookieWithNoID = true
					}
				}
			}

			if userID == "" && !cookieWithNoID {
				userID = generateUserID()
				token := signUserToken(userID, secretToUse)
				http.SetCookie(w, &http.Cookie{
					Name:     userCookieName,
					Value:    token,
					Path:     "/",
					HttpOnly: true,
				})
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, userID)
			ctx = context.WithValue(ctx, userCookieSeenContext, cookieWasPresent)
			ctx = context.WithValue(ctx, userCookieNoIDContext, cookieWithNoID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext возвращает идентификатор пользователя, установленный AuthMiddleware.
func UserIDFromContext(ctx context.Context) string {
	value := ctx.Value(userIDContextKey)
	if value == nil {
		return ""
	}
	userID, ok := value.(string)
	if !ok {
		return ""
	}
	return userID
}

// UserCookieWasPresent сообщает, была ли в запросе cookie user_token.
func UserCookieWasPresent(ctx context.Context) bool {
	value := ctx.Value(userCookieSeenContext)
	if value == nil {
		return false
	}
	wasPresent, ok := value.(bool)
	if !ok {
		return false
	}
	return wasPresent
}

// UserCookieHasNoID сообщает, что cookie была передана, но не содержала идентификатор пользователя.
func UserCookieHasNoID(ctx context.Context) bool {
	value := ctx.Value(userCookieNoIDContext)
	if value == nil {
		return false
	}
	hasNoID, ok := value.(bool)
	if !ok {
		return false
	}
	return hasNoID
}

func generateUserID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(buffer)
}

func signUserToken(userID string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(userID))
	signature := mac.Sum(nil)
	raw := userID + "." + hex.EncodeToString(signature)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func verifySignedUserToken(token string, secret string) (string, bool, bool) {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", false, false
	}

	parts := strings.SplitN(string(decoded), ".", 2)
	if len(parts) != 2 {
		return "", false, false
	}

	userID := parts[0]
	signatureHex := parts[1]

	signature, err := hex.DecodeString(signatureHex)
	if err != nil {
		return "", false, false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(userID))
	expected := mac.Sum(nil)
	if !hmac.Equal(signature, expected) {
		return "", false, false
	}

	return userID, true, userID != ""
}
