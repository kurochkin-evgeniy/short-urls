package app

import (
	"compress/flate"
	"net/http"
	"short-urls/internal/config"
	"short-urls/internal/handler"
	"short-urls/internal/logging"
	"short-urls/internal/service"

	"short-urls/internal/middleware"

	"github.com/go-chi/chi/v5"
	chi_m "github.com/go-chi/chi/v5/middleware"
)

type ShortUrlApp struct {
	cfg             *config.Config
	shortUrlService *service.ShortUrlService
}

func NewShortUrlApp() *ShortUrlApp {
	cfg := config.NewConfig()
	return &ShortUrlApp{
		shortUrlService: service.NewShortUrlService(cfg),
		cfg:             cfg,
	}
}

func (a *ShortUrlApp) Start() error {

	logging.LoggingInit()
	defer logging.LoggingDone()

	r := chi.NewRouter()

	compressor := chi_m.NewCompressor(flate.DefaultCompression, "/*")
	r.Use(compressor.Handler)

	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.DecompressRequestMiddleware)

	r.Post("/", handler.HandleCreateShortUrRequest(a.shortUrlService))
	r.Post("/api/shorten", handler.HandleCreateShortUrRequest(a.shortUrlService))
	r.Get("/{id}", handler.HandleRedirectRequest(a.shortUrlService))

	return http.ListenAndServe(a.cfg.HostAddr, r)
}
