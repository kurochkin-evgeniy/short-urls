// Package middleware предоставляет HTTP-middleware для логирования, аутентификации и распаковки.
package middleware

import (
	"compress/gzip"
	"context"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"

	"short-urls/internal/auth"
	"short-urls/internal/logging"
)

const userCookieName = "user_token"

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
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}
			defer gz.Close()

			r.Body = gz
			r.Header.Del("Content-Encoding")
			r.Header.Del("Content-Length")
		}
		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware назначает идентификатор пользователя через подписанную cookie и сохраняет его в контексте запроса.
func AuthMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(userCookieName)
			cookieWasPresent := err == nil

			userID := ""
			cookieWithNoID := false
			if cookieWasPresent {
				verifiedID, ok, hasID := auth.VerifySignedUserToken(cookie.Value, secret)
				if ok {
					userID = verifiedID
					if !hasID {
						cookieWithNoID = true
					}
				}
			}

			if userID == "" && !cookieWithNoID {
				userID = auth.GenerateUserID()
				token := auth.SignUserToken(userID, secret)
				http.SetCookie(w, &http.Cookie{
					Name:     userCookieName,
					Value:    token,
					Path:     "/",
					HttpOnly: true,
				})
			}

			ctx := auth.ContextWithUserID(r.Context(), userID, cookieWasPresent, cookieWithNoID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext возвращает идентификатор пользователя, установленный AuthMiddleware.
func UserIDFromContext(ctx context.Context) string {
	return auth.UserIDFromContext(ctx)
}

// UserCookieWasPresent сообщает, была ли в запросе cookie user_token.
func UserCookieWasPresent(ctx context.Context) bool {
	return auth.AuthWasPresent(ctx)
}

// UserCookieHasNoID сообщает, что cookie была передана, но не содержала идентификатор пользователя.
func UserCookieHasNoID(ctx context.Context) bool {
	return auth.AuthHasNoID(ctx)
}

// TrustedSubnetMiddleware разрешает запрос только если X-Real-IP входит в доверенную подсеть CIDR.
// При пустом trustedSubnet доступ запрещён для любого запроса.
func TrustedSubnetMiddleware(trustedSubnet string) func(http.Handler) http.Handler {
	var trustedNet *net.IPNet
	if trustedSubnet != "" {
		_, parsed, err := net.ParseCIDR(trustedSubnet)
		if err != nil {
			logging.Sugar.Errorf("failed to parse trusted subnet CIDR %q: %v", trustedSubnet, err)
		} else {
			trustedNet = parsed
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if trustedNet == nil {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			ip := net.ParseIP(r.Header.Get("X-Real-IP"))
			if ip == nil || !trustedNet.Contains(ip) {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
