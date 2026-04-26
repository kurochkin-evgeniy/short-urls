package app

import (
	"compress/flate"
	"database/sql"
	"fmt"
	"net/http"
	"short-urls/internal/config"
	"short-urls/internal/handler"
	"short-urls/internal/logging"
	"short-urls/internal/service"

	"short-urls/internal/middleware"

	"github.com/go-chi/chi/v5"
	chi_m "github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
)

type ShortUrlApp struct {
	cfg             *config.Config
	shortUrlService *service.ShortUrlService
}

func NewShortUrlApp() *ShortUrlApp {
	cfg := config.NewConfig()
	return &ShortUrlApp{
		cfg: cfg,
	}
}

func (a *ShortUrlApp) Start() error {

	logging.LoggingInit()
	defer logging.LoggingDone()

	if a.cfg.DatabaseDSN == "" {
		return fmt.Errorf("database dsn is empty: set DATABASE_DSN env or -d flag")
	}

	db, err := sql.Open("postgres", a.cfg.DatabaseDSN)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	a.shortUrlService = service.NewShortUrlService(a.cfg)

	r := chi.NewRouter()

	compressor := chi_m.NewCompressor(flate.DefaultCompression, "text/html", "application/json")
	r.Use(compressor.Handler)

	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.DecompressRequestMiddleware)

	r.Post("/", handler.HandleCreateShortUrRequest(a.shortUrlService))
	r.Post("/api/shorten", handler.HandleCreateShortUrRequest(a.shortUrlService))
	r.Get("/{id}", handler.HandleRedirectRequest(a.shortUrlService))
	r.Get("/ping", handler.HandlePing(db))

	return http.ListenAndServe(a.cfg.HostAddr, r)
}
