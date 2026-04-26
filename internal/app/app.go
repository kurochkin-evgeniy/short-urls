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
	logging.Sugar.Infow("Starting short URL service", "address", a.cfg.HostAddr, "base_url", a.cfg.BaseUrl)

	logging.Sugar.Debugw("Building storage backend")
	storage, db, err := a.buildStorage()
	if err != nil {
		logging.Sugar.Errorw("Failed to build storage backend", "error", err)
		return err
	}
	if db != nil {
		logging.Sugar.Debugw("Database connection opened, close deferred")
		defer db.Close()
	}

	logging.Sugar.Debugw("Initializing short URL service")
	a.shortUrlService = service.NewShortUrlService(a.cfg.BaseUrl, storage)

	logging.Sugar.Debugw("Configuring HTTP router")
	r := chi.NewRouter()

	compressor := chi_m.NewCompressor(flate.DefaultCompression, "text/html", "application/json")
	r.Use(compressor.Handler)

	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.DecompressRequestMiddleware)

	r.Post("/", handler.HandleCreateShortUrRequest(a.shortUrlService))
	r.Post("/api/shorten", handler.HandleCreateShortUrRequest(a.shortUrlService))
	r.Post("/api/shorten/batch", handler.HandleCreateBatchShortUrRequest(a.shortUrlService))
	r.Get("/{id}", handler.HandleRedirectRequest(a.shortUrlService))
	r.Get("/ping", handler.HandlePing(db))

	logging.Sugar.Infow("HTTP server is starting", "address", a.cfg.HostAddr)
	if err := http.ListenAndServe(a.cfg.HostAddr, r); err != nil {
		logging.Sugar.Errorw("HTTP server stopped with error", "error", err)
		return err
	}

	logging.Sugar.Infow("HTTP server stopped")
	return nil
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
		if err := repository.RunPostgresMigrations(a.cfg.DatabaseDSN); err != nil {
			db.Close()
			logging.Sugar.Errorw("Failed to run PostgreSQL migrations", "error", err)
			return nil, nil, fmt.Errorf("run migrations: %w", err)
		}
		logging.Sugar.Infow("PostgreSQL migrations completed")
		logging.Sugar.Debugw("Pinging PostgreSQL after migrations")
		if err := db.Ping(); err != nil {
			db.Close()
			logging.Sugar.Errorw("Failed to ping PostgreSQL after migrations", "error", err)
			return nil, nil, fmt.Errorf("ping database after migrations: %w", err)
		}
		logging.Sugar.Infow("PostgreSQL is reachable after migrations")

		storage := repository.NewPostgresKeyValueStorage(db)
		return storage, db, nil
	}

	if a.cfg.FilePath != "" {
		logging.Sugar.Infow("Using file storage", "path", a.cfg.FilePath)
		return repository.NewMapKeyValuePermanentStorage(a.cfg.FilePath), nil, nil
	}

	logging.Sugar.Infow("Using in-memory storage")
	return repository.NewMapKeyValueStorage(), nil, nil
}
