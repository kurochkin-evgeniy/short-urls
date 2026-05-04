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

func DecompressRequestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") == "gzip" {
			// Wrap the request body with a gzip reader
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}
			defer gz.Close()

			// Replace the original request body with the decompressed one
			r.Body = gz
			// Also, update the Content-Length and remove Content-Encoding headers
			r.Header.Del("Content-Encoding")
			r.Header.Del("Content-Length")
		}
		// Continue to the next handler
		next.ServeHTTP(w, r)
	})
}

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
