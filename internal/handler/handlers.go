// Package handler предоставляет HTTP-обработчики эндпоинтов сервиса сокращения URL.
package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"short-urls/internal/logging"
	"short-urls/internal/middleware"
	"short-urls/internal/service"
	"time"
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
	OriginalURL   string `json:"original_url"`
}

// BatchShortUrlResponse описывает один элемент пакетного ответа.
type BatchShortUrlResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// UserURLResponse описывает пару URL, возвращаемую GET /api/user/urls.
type UserURLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func handleCreateShortUrl(responce http.ResponseWriter, request *http.Request, s *service.ShortUrlService) {
	userID := middleware.UserIDFromContext(request.Context())

	body, err := io.ReadAll(request.Body)
	if err == nil {

		switch request.Header.Get("content-type") {
		case "text/plain":
			{
				originalURL := string(body)
				createResult, err := s.CreateShortUrl(originalURL, userID)
				if err != nil {
					logging.Sugar.Errorw("Failed to create short url", "error", err)
					responce.WriteHeader(http.StatusInternalServerError)
					return
				}
				responce.Header().Set("content-type", "text/plain")
				if createResult.WasInserted {
					responce.WriteHeader(http.StatusCreated)
				} else {
					responce.WriteHeader(http.StatusConflict)
				}
				responce.Write([]byte(createResult.ShortURL))
				s.AuditShorten(originalURL, userID)
				return
			}
		case "application/json":
			{
				var r CreateShortUrlRequest
				if err := json.Unmarshal(body, &r); err == nil {
					createResult, err := s.CreateShortUrl(r.Url, userID)
					if err != nil {
						logging.Sugar.Errorw("Failed to create short url", "error", err)
						responce.WriteHeader(http.StatusInternalServerError)
						return
					}

					var resp CreateShortUrlResponse
					resp.Result = createResult.ShortURL

					respStr, err := json.Marshal(resp)
					if err != nil {
						responce.WriteHeader(http.StatusInternalServerError)
						return
					}

					responce.Header().Set("content-type", "application/json")
					if createResult.WasInserted {
						responce.WriteHeader(http.StatusCreated)
					} else {
						responce.WriteHeader(http.StatusConflict)
					}
					responce.Write(respStr)
					s.AuditShorten(r.Url, userID)
				}
			}
		}
	}

	responce.WriteHeader(http.StatusBadRequest)
}

func handleCreateBatchShortUrl(responce http.ResponseWriter, request *http.Request, s *service.ShortUrlService) {
	userID := middleware.UserIDFromContext(request.Context())

	body, err := io.ReadAll(request.Body)
	if err != nil {
		responce.WriteHeader(http.StatusBadRequest)
		return
	}

	var batchReq []BatchShortUrlRequest
	if err := json.Unmarshal(body, &batchReq); err != nil {
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

	createResults, err := s.CreateBatchShortUrls(urls, userID)
	if err != nil {
		logging.Sugar.Errorw("Failed to create batch short urls", "error", err)
		responce.WriteHeader(http.StatusInternalServerError)
		return
	}
	batchResp := make([]BatchShortUrlResponse, 0, len(batchReq))
	for i, item := range batchReq {
		batchResp = append(batchResp, BatchShortUrlResponse{
			CorrelationID: item.CorrelationID,
			ShortURL:      createResults[i],
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

func handleGetUserURLs(responce http.ResponseWriter, request *http.Request, s *service.ShortUrlService) {
	userID := middleware.UserIDFromContext(request.Context())
	if middleware.UserCookieWasPresent(request.Context()) && middleware.UserCookieHasNoID(request.Context()) {
		responce.WriteHeader(http.StatusUnauthorized)
		return
	}

	userURLs, err := s.GetUserURLs(userID)
	if err != nil {
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
			ShortURL:    item.ShortURL,
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

func handleRedirectUrl(responce http.ResponseWriter, request *http.Request, s *service.ShortUrlService) {

	id := request.URL.Path[1:]
	url, isDeleted, found := s.LookupShortURL(id)
	if !found {
		responce.WriteHeader(http.StatusBadRequest)
		return
	}
	if isDeleted {
		responce.WriteHeader(http.StatusGone)
		return
	}
	responce.Header().Add("Location", url)
	responce.WriteHeader(http.StatusTemporaryRedirect)
	s.AuditFollow(url, middleware.UserIDFromContext(request.Context()))
}

func handleDeleteUserURLs(responce http.ResponseWriter, request *http.Request, s *service.ShortUrlService) {
	userID := middleware.UserIDFromContext(request.Context())
	if middleware.UserCookieWasPresent(request.Context()) && middleware.UserCookieHasNoID(request.Context()) {
		responce.WriteHeader(http.StatusUnauthorized)
		return
	}

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

	s.QueueUserURLsDeletion(userID, shortIDs)
	responce.WriteHeader(http.StatusAccepted)
}

// HandleRedirectRequest обрабатывает GET /{id} и перенаправляет на оригинальный URL.
func HandleRedirectRequest(s *service.ShortUrlService) http.HandlerFunc {
	return func(responce http.ResponseWriter, request *http.Request) {
		handleRedirectUrl(responce, request, s)
	}
}

// HandleCreateShortUrRequest обрабатывает POST / и POST /api/shorten.
func HandleCreateShortUrRequest(s *service.ShortUrlService) http.HandlerFunc {
	return func(responce http.ResponseWriter, request *http.Request) {
		handleCreateShortUrl(responce, request, s)
	}
}

// HandleCreateBatchShortUrRequest обрабатывает POST /api/shorten/batch.
func HandleCreateBatchShortUrRequest(s *service.ShortUrlService) http.HandlerFunc {
	return func(responce http.ResponseWriter, request *http.Request) {
		handleCreateBatchShortUrl(responce, request, s)
	}
}

// HandleGetUserURLsRequest обрабатывает GET /api/user/urls.
func HandleGetUserURLsRequest(s *service.ShortUrlService) http.HandlerFunc {
	return func(responce http.ResponseWriter, request *http.Request) {
		handleGetUserURLs(responce, request, s)
	}
}

// HandleDeleteUserURLsRequest обрабатывает DELETE /api/user/urls.
func HandleDeleteUserURLsRequest(s *service.ShortUrlService) http.HandlerFunc {
	return func(responce http.ResponseWriter, request *http.Request) {
		handleDeleteUserURLs(responce, request, s)
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
