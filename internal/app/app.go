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
	mux.HandleFunc("/{id}", handler.HandleRequest(a.shortUrlService))
	mux.HandleFunc("/", handler.HandleRequest(a.shortUrlService))

	return http.ListenAndServe(":8080", mux)
}
