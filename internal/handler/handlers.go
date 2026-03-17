package handler

import (
	"io"
	"net/http"
	"short-urls/internal/service"
)

func handleCreateShortUrl(responce http.ResponseWriter, request *http.Request, s *service.ShortUrlService) {

	ct := request.Header.Get("content-type")
	if ct == "text/plain" {
		body, err := io.ReadAll(request.Body)
		if err == nil {
			shortUrl := s.CreateShortUrl(string(body))
			responce.Header().Set("content-type", "text/plain")
			responce.WriteHeader(http.StatusCreated)
			responce.Write([]byte(shortUrl))
			return
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
