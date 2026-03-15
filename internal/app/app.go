package app

import (
	"net/http"
	"short-urls/internal/handler"
	"short-urls/internal/service"
)

type ShortUrlApp struct {
	shortUrlService *service.ShortUrlService
}

func NewShortUrlApp() *ShortUrlApp {
	return &ShortUrlApp{shortUrlService: service.NewShortUrlService()}
}

func (a *ShortUrlApp) Start() error {

	mux := http.NewServeMux()
	mux.HandleFunc("/{id}", func(w http.ResponseWriter, r *http.Request) { handler.HandleRequest(w, r, a.shortUrlService) })
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { handler.HandleRequest(w, r, a.shortUrlService) })

	return http.ListenAndServe(":8080", mux)
}
