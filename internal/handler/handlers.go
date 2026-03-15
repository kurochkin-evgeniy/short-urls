package handler

import (
	"io"
	"net/http"
	"short-urls/internal/service"
)

func handleCreateShortUrl(responce http.ResponseWriter, request *http.Request, s *service.ShortUrlService) {

	ct := request.Header.Get("content-type")
	if ct == "text/plain" && request.URL.Path == "/" {
		body, err := io.ReadAll(request.Body)
		if err == nil {
			id := s.CreateShortUrl(string(body))
			responce.WriteHeader(http.StatusCreated)
			responce.Write([]byte("http://localhost:8080/" + id))
			return
		}
	}

	responce.WriteHeader(http.StatusBadRequest)
}

func handleRedirectUrl(responce http.ResponseWriter, request *http.Request, s *service.ShortUrlService) {

	id := request.PathValue("id")
	url := s.ResolveShortUrl(id)
	if url != "" {
		responce.Header().Add("Location", url)
		responce.WriteHeader(http.StatusTemporaryRedirect)
		return
	}

	responce.WriteHeader(http.StatusBadRequest)
}

func HandleRequest(responce http.ResponseWriter, request *http.Request, s *service.ShortUrlService) {
	switch request.Method {
	case http.MethodPost:
		handleCreateShortUrl(responce, request, s)
	case http.MethodGet:
		handleRedirectUrl(responce, request, s)
	default:
		responce.WriteHeader(http.StatusBadRequest)
	}
}
