package app

import (
	"compress/flate"
	"database/sql"
	"fmt"
	"net/http"
	"short-urls/internal/config"
	"short-urls/internal/handler"
	"short-urls/internal/logging"
	"short-urls/internal/repository"
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

	storage, db, err := a.buildStorage()
	if err != nil {
		return err
	}
	if db != nil {
		defer db.Close()
	}

	a.shortUrlService = service.NewShortUrlService(a.cfg.BaseUrl, storage)

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

func (a *ShortUrlApp) buildStorage() (repository.KeyValueStorage, *sql.DB, error) {
	if a.cfg.DatabaseDSN != "" {
		logging.Sugar.Infow("Using PostgreSQL storage")
		logging.Sugar.Debugw("Opening PostgreSQL connection")
		db, err := sql.Open("postgres", a.cfg.DatabaseDSN)
		if err != nil {
			logging.Sugar.Errorw("Failed to open PostgreSQL connection", "error", err)
			return nil, nil, fmt.Errorf("open database: %w", err)
		}

		logging.Sugar.Debugw("Pinging PostgreSQL")
		if err := db.Ping(); err != nil {
			db.Close()
			logging.Sugar.Errorw("Failed to ping PostgreSQL", "error", err)
			return nil, nil, fmt.Errorf("ping database: %w", err)
		}
		logging.Sugar.Infow("PostgreSQL connection established")

		logging.Sugar.Infow("Running PostgreSQL migrations")
		storage, err := repository.NewPostgresKeyValueStorage(db)
		if err != nil {
			db.Close()
			logging.Sugar.Errorw("Failed to run PostgreSQL migrations", "error", err)
			return nil, nil, fmt.Errorf("run migrations: %w", err)
		}
		logging.Sugar.Infow("PostgreSQL migrations completed")

		return storage, db, nil
	}

	if a.cfg.FilePath != "" {
		logging.Sugar.Infow("Using file storage", "path", a.cfg.FilePath)
		return repository.NewMapKeyValuePermanentStorage(a.cfg.FilePath), nil, nil
	}

	logging.Sugar.Infow("Using in-memory storage")
	return repository.NewMapKeyValueStorage(), nil, nil
}
