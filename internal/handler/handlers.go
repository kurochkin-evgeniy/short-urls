package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"short-urls/internal/service"
)

type CreateShortUrlRequest struct {
	Url string `json:"url,omitempty"`
}

type CreateShortUrlResponse struct {
	Result string `json:"result,omitempty"`
}

func handleCreateShortUrl(responce http.ResponseWriter, request *http.Request, s *service.ShortUrlService) {

	body, err := io.ReadAll(request.Body)
	if err == nil {

		switch request.Header.Get("content-type") {
		case "text/plain":
			{
				shortUrl := s.CreateShortUrl(string(body))
				responce.Header().Set("content-type", "text/plain")
				responce.WriteHeader(http.StatusCreated)
				responce.Write([]byte(shortUrl))
				return
			}
		case "application/json":
			{
				var r CreateShortUrlRequest
				if err := json.Unmarshal(body, &r); err == nil {

					var resp CreateShortUrlResponse
					resp.Result = s.CreateShortUrl(r.Url)

					respStr, err := json.Marshal(resp)
					if err != nil {
						responce.WriteHeader(http.StatusInternalServerError)
						return
					}

					responce.Header().Set("content-type", "application/json")
					responce.WriteHeader(http.StatusCreated)
					responce.Write(respStr)
				}
			}
		}
	}

	responce.WriteHeader(http.StatusBadRequest)
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
