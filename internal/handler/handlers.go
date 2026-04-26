package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"short-urls/internal/logging"
	"short-urls/internal/service"
	"time"
)

type CreateShortUrlRequest struct {
	Url string `json:"url,omitempty"`
}

type CreateShortUrlResponse struct {
	Result string `json:"result,omitempty"`
}

type BatchShortUrlRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type BatchShortUrlResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

func handleCreateShortUrl(responce http.ResponseWriter, request *http.Request, s *service.ShortUrlService) {

	body, err := io.ReadAll(request.Body)
	if err == nil {

		switch request.Header.Get("content-type") {
		case "text/plain":
			{
				createResult := s.CreateShortUrl(string(body))
				responce.Header().Set("content-type", "text/plain")
				if createResult.WasInserted {
					responce.WriteHeader(http.StatusCreated)
				} else {
					responce.WriteHeader(http.StatusConflict)
				}
				responce.Write([]byte(createResult.ShortURL))
				return
			}
		case "application/json":
			{
				var r CreateShortUrlRequest
				if err := json.Unmarshal(body, &r); err == nil {
					createResult := s.CreateShortUrl(r.Url)

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
				}
			}
		}
	}

	responce.WriteHeader(http.StatusBadRequest)
}

func handleCreateBatchShortUrl(responce http.ResponseWriter, request *http.Request, s *service.ShortUrlService) {
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

	batchResp := make([]BatchShortUrlResponse, 0, len(batchReq))
	for _, item := range batchReq {
		createResult := s.CreateShortUrl(item.OriginalURL)
		batchResp = append(batchResp, BatchShortUrlResponse{
			CorrelationID: item.CorrelationID,
			ShortURL:      createResult.ShortURL,
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

func handleRedirectUrl(responce http.ResponseWriter, request *http.Request, s *service.ShortUrlService) {

	id := request.URL.Path[1:]
	url := s.ResolveShortUrl(id)
	if url != "" {
		responce.Header().Add("Location", url)
		responce.WriteHeader(http.StatusTemporaryRedirect)
		return
	}

	responce.WriteHeader(http.StatusBadRequest)
}

func HandleRedirectRequest(s *service.ShortUrlService) http.HandlerFunc {
	return func(responce http.ResponseWriter, request *http.Request) {
		handleRedirectUrl(responce, request, s)
	}
}

func HandleCreateShortUrRequest(s *service.ShortUrlService) http.HandlerFunc {
	return func(responce http.ResponseWriter, request *http.Request) {
		handleCreateShortUrl(responce, request, s)
	}
}

func HandleCreateBatchShortUrRequest(s *service.ShortUrlService) http.HandlerFunc {
	return func(responce http.ResponseWriter, request *http.Request) {
		handleCreateBatchShortUrl(responce, request, s)
	}
}

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
