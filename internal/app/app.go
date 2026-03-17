package app

import (
	"net/http"
	"short-urls/internal/config"
	"short-urls/internal/handler"
	"short-urls/internal/service"

	"github.com/go-chi/chi/v5"
)

type ShortUrlApp struct {
	cfg             *config.Config
	shortUrlService *service.ShortUrlService
}

func NewShortUrlApp() *ShortUrlApp {
	cfg := config.NewConfig()
	return &ShortUrlApp{
		shortUrlService: service.NewShortUrlService(cfg.BaseUrl),
		cfg:             cfg,
	}
}

func (a *ShortUrlApp) Start() error {

	r := chi.NewRouter()

	r.Post("/", handler.HandleCreateShortUrRequest(a.shortUrlService))
	r.Get("/{id}", handler.HandleRedirectRequest(a.shortUrlService))

	return http.ListenAndServe(a.cfg.HostAddr, r)
}
