package middleware

import (
	"net/http"
	"short-urls/internal/logging"
	"time"

	"github.com/go-chi/chi/middleware"
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
