package middleware

import (
	"compress/gzip"
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
