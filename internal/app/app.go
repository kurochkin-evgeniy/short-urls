package app

import (
	"net/http"
	"short-urls/internal/config"
	"short-urls/internal/handler"
	"short-urls/internal/logging"
	"short-urls/internal/service"

	"short-urls/internal/middleware"

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

	logging.LoggingInit()
	defer logging.LoggingDone()

	r := chi.NewRouter()

	r.Use(middleware.LoggingMiddleware)
	r.Post("/", handler.HandleCreateShortUrRequest(a.shortUrlService))
	r.Post("/api/shorten", handler.HandleCreateShortUrRequest(a.shortUrlService))
	r.Get("/{id}", handler.HandleRedirectRequest(a.shortUrlService))

	return http.ListenAndServe(a.cfg.HostAddr, r)
}
