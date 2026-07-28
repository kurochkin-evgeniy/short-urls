// Package handler предоставляет HTTP-обработчики эндпоинтов сервиса сокращения URL.

package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"short-urls/internal/facade"
	"short-urls/internal/logging"
)

// CreateShortUrlRequest — тело JSON-запроса для POST /api/shorten.

type CreateShortUrlRequest struct {
	Url string `json:"url,omitempty"`
}

// CreateShortUrlResponse — тело JSON-ответа для POST /api/shorten.

type CreateShortUrlResponse struct {
	Result string `json:"result,omitempty"`
}

// BatchShortUrlRequest описывает один элемент пакетного запроса на сокращение.

type BatchShortUrlRequest struct {
	CorrelationID string `json:"correlation_id"`

	OriginalURL string `json:"original_url"`
}

// BatchShortUrlResponse описывает один элемент пакетного ответа.

type BatchShortUrlResponse struct {
	CorrelationID string `json:"correlation_id"`

	ShortURL string `json:"short_url"`
}

// UserURLResponse описывает пару URL, возвращаемую GET /api/user/urls.

type UserURLResponse struct {
	ShortURL string `json:"short_url"`

	OriginalURL string `json:"original_url"`
}

// StatsResponse — тело JSON-ответа для GET /api/internal/stats.

type StatsResponse struct {
	URLs int `json:"urls"`

	Users int `json:"users"`
}

func handleCreateShortUrl(responce http.ResponseWriter, request *http.Request, f *facade.Shortener) {

	body, err := io.ReadAll(request.Body)

	if err == nil {

		switch request.Header.Get("content-type") {

		case "text/plain":

			originalURL := string(body)

			shortURL, wasInserted, err := f.ShortenURL(request.Context(), originalURL)

			if err != nil {

				logging.Sugar.Errorw("Failed to create short url", "error", err)

				responce.WriteHeader(http.StatusInternalServerError)

				return

			}

			responce.Header().Set("content-type", "text/plain")

			if wasInserted {

				responce.WriteHeader(http.StatusCreated)

			} else {

				responce.WriteHeader(http.StatusConflict)

			}

			responce.Write([]byte(shortURL))

			return

		case "application/json":

			var r CreateShortUrlRequest

			if err := json.Unmarshal(body, &r); err == nil {

				shortURL, wasInserted, err := f.ShortenURL(request.Context(), r.Url)

				if err != nil {

					logging.Sugar.Errorw("Failed to create short url", "error", err)

					responce.WriteHeader(http.StatusInternalServerError)

					return

				}

				var resp CreateShortUrlResponse

				resp.Result = shortURL

				respStr, err := json.Marshal(resp)

				if err != nil {

					responce.WriteHeader(http.StatusInternalServerError)

					return

				}

				responce.Header().Set("content-type", "application/json")

				if wasInserted {

					responce.WriteHeader(http.StatusCreated)

				} else {

					responce.WriteHeader(http.StatusConflict)

				}

				responce.Write(respStr)

			}

		}

	}

	responce.WriteHeader(http.StatusBadRequest)

}

func handleCreateBatchShortUrl(responce http.ResponseWriter, request *http.Request, f *facade.Shortener) {

	body, err := io.ReadAll(request.Body)

	if err != nil {

		responce.WriteHeader(http.StatusBadRequest)

		return

	}

	var batchReq []BatchShortUrlRequest

	if unmarshalErr := json.Unmarshal(body, &batchReq); unmarshalErr != nil {

		responce.WriteHeader(http.StatusBadRequest)

		return

	}

	if len(batchReq) == 0 {

		responce.WriteHeader(http.StatusBadRequest)

		return

	}

	urls := make([]string, 0, len(batchReq))

	for _, item := range batchReq {

		urls = append(urls, item.OriginalURL)

	}

	createResults, err := f.CreateBatchShortUrls(request.Context(), urls)

	if err != nil {

		logging.Sugar.Errorw("Failed to create batch short urls", "error", err)

		responce.WriteHeader(http.StatusInternalServerError)

		return

	}

	batchResp := make([]BatchShortUrlResponse, 0, len(batchReq))

	for i, item := range batchReq {

		batchResp = append(batchResp, BatchShortUrlResponse{

			CorrelationID: item.CorrelationID,

			ShortURL: createResults[i],
		})

	}

	respBody, err := json.Marshal(batchResp)

	if err != nil {

		responce.WriteHeader(http.StatusInternalServerError)

		return

	}

	responce.Header().Set("content-type", "application/json")

	responce.WriteHeader(http.StatusCreated)

	responce.Write(respBody)

}

func handleGetUserURLs(responce http.ResponseWriter, request *http.Request, f *facade.Shortener) {

	userURLs, err := f.ListUserURLs(request.Context())

	if err != nil {

		if errors.Is(err, facade.ErrUnauthorized) {

			responce.WriteHeader(http.StatusUnauthorized)

			return

		}

		logging.Sugar.Errorw("Failed to read user urls", "error", err)

		responce.WriteHeader(http.StatusInternalServerError)

		return

	}

	if len(userURLs) == 0 {

		responce.WriteHeader(http.StatusNoContent)

		return

	}

	result := make([]UserURLResponse, 0, len(userURLs))

	for _, item := range userURLs {

		result = append(result, UserURLResponse{

			ShortURL: item.ShortURL,

			OriginalURL: item.OriginalURL,
		})

	}

	body, err := json.Marshal(result)

	if err != nil {

		responce.WriteHeader(http.StatusInternalServerError)

		return

	}

	responce.Header().Set("content-type", "application/json")

	responce.WriteHeader(http.StatusOK)

	responce.Write(body)

}

func handleRedirectUrl(responce http.ResponseWriter, request *http.Request, f *facade.Shortener) {

	id := request.URL.Path[1:]

	url, err := f.ExpandURL(request.Context(), id)

	if err != nil {

		switch {

		case errors.Is(err, facade.ErrNotFound):

			responce.WriteHeader(http.StatusBadRequest)

		case errors.Is(err, facade.ErrGone):

			responce.WriteHeader(http.StatusGone)

		default:

			responce.WriteHeader(http.StatusInternalServerError)

		}

		return

	}

	responce.Header().Add("Location", url)

	responce.WriteHeader(http.StatusTemporaryRedirect)

}

func handleDeleteUserURLs(responce http.ResponseWriter, request *http.Request, f *facade.Shortener) {

	body, err := io.ReadAll(request.Body)

	if err != nil {

		responce.WriteHeader(http.StatusBadRequest)

		return

	}

	var shortIDs []string

	if err := json.Unmarshal(body, &shortIDs); err != nil {

		responce.WriteHeader(http.StatusBadRequest)

		return

	}

	if len(shortIDs) == 0 {

		responce.WriteHeader(http.StatusBadRequest)

		return

	}

	if err := f.QueueUserURLsDeletion(request.Context(), shortIDs); err != nil {

		if errors.Is(err, facade.ErrUnauthorized) {

			responce.WriteHeader(http.StatusUnauthorized)

			return

		}

		responce.WriteHeader(http.StatusInternalServerError)

		return

	}

	responce.WriteHeader(http.StatusAccepted)

}

// HandleRedirectRequest обрабатывает GET /{id} и перенаправляет на оригинальный URL.

func HandleRedirectRequest(f *facade.Shortener) http.HandlerFunc {

	return func(responce http.ResponseWriter, request *http.Request) {

		handleRedirectUrl(responce, request, f)

	}

}

// HandleCreateShortUrRequest обрабатывает POST / и POST /api/shorten.

func HandleCreateShortUrRequest(f *facade.Shortener) http.HandlerFunc {

	return func(responce http.ResponseWriter, request *http.Request) {

		handleCreateShortUrl(responce, request, f)

	}

}

// HandleCreateBatchShortUrRequest обрабатывает POST /api/shorten/batch.

func HandleCreateBatchShortUrRequest(f *facade.Shortener) http.HandlerFunc {

	return func(responce http.ResponseWriter, request *http.Request) {

		handleCreateBatchShortUrl(responce, request, f)

	}

}

// HandleGetUserURLsRequest обрабатывает GET /api/user/urls.

func HandleGetUserURLsRequest(f *facade.Shortener) http.HandlerFunc {

	return func(responce http.ResponseWriter, request *http.Request) {

		handleGetUserURLs(responce, request, f)

	}

}

// HandleDeleteUserURLsRequest обрабатывает DELETE /api/user/urls.

func HandleDeleteUserURLsRequest(f *facade.Shortener) http.HandlerFunc {

	return func(responce http.ResponseWriter, request *http.Request) {

		handleDeleteUserURLs(responce, request, f)

	}

}

// HandlePing обрабатывает GET /ping и проверяет доступность PostgreSQL.

func HandlePing(db *sql.DB) http.HandlerFunc {

	return func(responce http.ResponseWriter, request *http.Request) {

		if db == nil {

			logging.Sugar.Warnw("Ping requested but database is not configured")

			responce.WriteHeader(http.StatusInternalServerError)

			return

		}

		logging.Sugar.Debugw("Checking PostgreSQL health with ping")

		ctx, cancel := context.WithTimeout(request.Context(), 2*time.Second)

		defer cancel()

		if err := db.PingContext(ctx); err != nil {

			logging.Sugar.Errorw("PostgreSQL ping failed", "error", err)

			responce.WriteHeader(http.StatusInternalServerError)

			return

		}

		logging.Sugar.Debugw("PostgreSQL ping succeeded")

		responce.WriteHeader(http.StatusOK)

	}

}

func handleGetStats(responce http.ResponseWriter, request *http.Request, f *facade.Shortener) {

	stats, err := f.GetStats(request.Context())

	if err != nil {

		logging.Sugar.Errorw("Failed to read stats", "error", err)

		responce.WriteHeader(http.StatusInternalServerError)

		return

	}

	body, err := json.Marshal(StatsResponse{

		URLs: stats.URLs,

		Users: stats.Users,
	})

	if err != nil {

		responce.WriteHeader(http.StatusInternalServerError)

		return

	}

	responce.Header().Set("content-type", "application/json")

	responce.WriteHeader(http.StatusOK)

	responce.Write(body)

}

// HandleGetStatsRequest обрабатывает GET /api/internal/stats.

func HandleGetStatsRequest(f *facade.Shortener) http.HandlerFunc {

	return func(responce http.ResponseWriter, request *http.Request) {

		handleGetStats(responce, request, f)

	}

}
