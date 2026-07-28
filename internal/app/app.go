// Package app связывает конфигурацию, хранилище, обработчики и HTTP/gRPC-серверы.
package app

import (
	"compress/flate"
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chi_m "github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"

	"short-urls/internal/audit"
	"short-urls/internal/config"
	"short-urls/internal/facade"
	"short-urls/internal/grpchandler"
	"short-urls/internal/grpcmiddleware"
	"short-urls/internal/handler"
	"short-urls/internal/logging"
	"short-urls/internal/middleware"
	"short-urls/internal/repository"
	"short-urls/internal/service"
	"short-urls/pkg/shortenerv1"
)

const shutdownTimeout = 30 * time.Second

// ShortUrlApp — корневой объект приложения.
type ShortUrlApp struct {
	cfg             *config.Config
	shortUrlService *service.ShortUrlService
	shortenerFacade *facade.Shortener
	srv             *http.Server
	grpcSrv         *grpc.Server
	listener        net.Listener
	srvMux          cmux.CMux
	storage         repository.KeyValueStorage
	auditSubject    *audit.Subject
	db              *sql.DB
}

// NewShortUrlApp загружает конфигурацию и подготавливает экземпляр приложения.
func NewShortUrlApp() (*ShortUrlApp, error) {
	cfg, err := config.NewConfig()
	if err != nil {
		return nil, err
	}
	return &ShortUrlApp{
		cfg: cfg,
	}, nil
}

// Start собирает зависимости, регистрирует маршруты и запускает HTTP/gRPC-серверы.
func (a *ShortUrlApp) Start() error {
	logging.LoggingInit()
	defer logging.LoggingDone()
	logging.Sugar.Infow("Starting short URL service", "address", a.cfg.HostAddr, "base_url", a.cfg.BaseUrl)

	go func() {
		logging.Sugar.Infow("Starting pprof server", "address", "localhost:6060")
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			logging.Sugar.Errorw("pprof server stopped with error", "error", err)
		}
	}()

	logging.Sugar.Debugw("Building storage backend")
	storage, db, err := a.buildStorage()
	if err != nil {
		logging.Sugar.Errorw("Failed to build storage backend", "error", err)
		return err
	}
	a.storage = storage
	a.db = db
	if db != nil {
		logging.Sugar.Debugw("Database connection opened")
	}

	logging.Sugar.Debugw("Initializing short URL service")
	a.auditSubject = audit.NewSubjectFromConfig(a.cfg.AuditFile, a.cfg.AuditURL)
	a.shortUrlService = service.NewShortUrlService(a.cfg.BaseUrl, a.storage, a.auditSubject)
	a.shortenerFacade = facade.New(a.shortUrlService)

	logging.Sugar.Debugw("Configuring HTTP router")
	r := chi.NewRouter()

	compressor := chi_m.NewCompressor(flate.DefaultCompression, "text/html", "application/json")
	r.Use(compressor.Handler)

	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.DecompressRequestMiddleware)
	r.Use(middleware.AuthMiddleware(a.cfg.CookieSecret))

	r.Post("/", handler.HandleCreateShortUrRequest(a.shortenerFacade))
	r.Post("/api/shorten", handler.HandleCreateShortUrRequest(a.shortenerFacade))
	r.Post("/api/shorten/batch", handler.HandleCreateBatchShortUrRequest(a.shortenerFacade))
	r.Get("/api/user/urls", handler.HandleGetUserURLsRequest(a.shortenerFacade))
	r.Delete("/api/user/urls", handler.HandleDeleteUserURLsRequest(a.shortenerFacade))
	r.With(middleware.TrustedSubnetMiddleware(a.cfg.TrustedSubnet)).Get(
		"/api/internal/stats",
		handler.HandleGetStatsRequest(a.shortenerFacade),
	)
	r.Get("/{id}", handler.HandleRedirectRequest(a.shortenerFacade))
	r.Get("/ping", handler.HandlePing(a.db))

	a.srv = &http.Server{
		Handler: r,
	}

	a.grpcSrv = grpc.NewServer(
		grpc.UnaryInterceptor(grpcmiddleware.AuthUnaryInterceptor(a.cfg.CookieSecret)),
	)
	shortenerv1.RegisterShortenerServiceServer(a.grpcSrv, grpchandler.NewShortenerServer(a.shortenerFacade))

	listener, err := net.Listen("tcp", a.cfg.HostAddr)
	if err != nil {
		logging.Sugar.Errorw("Failed to start listener", "error", err)
		return fmt.Errorf("listen: %w", err)
	}
	if a.cfg.EnableHTTPS {
		cert, err := tls.LoadX509KeyPair(a.cfg.TLSCertFile, a.cfg.TLSKeyFile)
		if err != nil {
			listener.Close()
			return fmt.Errorf("load tls certificate: %w", err)
		}
		listener = tls.NewListener(listener, &tls.Config{
			Certificates: []tls.Certificate{cert},
			NextProtos:   []string{"h2", "http/1.1"},
		})
	}

	a.listener = listener
	a.srvMux = cmux.New(listener)
	grpcListener := a.srvMux.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	httpListener := a.srvMux.Match(cmux.Any())

	serverErrors := make(chan error, 3)
	go func() {
		if err := a.grpcSrv.Serve(grpcListener); err != nil {
			serverErrors <- err
		}
	}()
	go func() {
		if err := a.srv.Serve(httpListener); err != nil && err != http.ErrServerClosed && err != cmux.ErrListenerClosed {
			serverErrors <- err
		}
	}()
	go func() {
		if a.cfg.EnableHTTPS {
			logging.Sugar.Infow("HTTPS/gRPC server is starting",
				"address", a.cfg.HostAddr,
				"cert", a.cfg.TLSCertFile,
				"key", a.cfg.TLSKeyFile,
			)
		} else {
			logging.Sugar.Infow("HTTP/gRPC server is starting", "address", a.cfg.HostAddr)
		}
		if err := a.srvMux.Serve(); err != nil && err != cmux.ErrListenerClosed {
			serverErrors <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	var runErr error
	select {
	case runErr = <-serverErrors:
		logging.Sugar.Errorw("Server stopped with error", "error", runErr)
	case sig := <-quit:
		logging.Sugar.Infow("Shutdown signal received", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := a.Shutdown(ctx); err != nil && runErr == nil {
		return err
	}
	return runErr
}

// Shutdown останавливает HTTP/gRPC-серверы и освобождает ресурсы приложения.
func (a *ShortUrlApp) Shutdown(ctx context.Context) error {
	if a.grpcSrv != nil {
		a.grpcSrv.GracefulStop()
	}

	if a.srv != nil {
		if err := a.srv.Shutdown(ctx); err != nil {
			logging.Sugar.Errorw("Server graceful shutdown failed", "error", err)
			return err
		}
	}

	if a.srvMux != nil {
		a.srvMux.Close()
	}
	if a.listener != nil {
		a.listener.Close()
	}

	logging.Sugar.Infow("HTTP/gRPC servers stopped, draining background workers")
	if a.shortUrlService != nil {
		a.shortUrlService.Shutdown()
	}

	if a.auditSubject != nil {
		if err := a.auditSubject.Close(); err != nil {
			logging.Sugar.Errorw("Failed to close audit", "error", err)
		}
	}

	if closer, ok := a.storage.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			logging.Sugar.Errorw("Failed to close storage", "error", err)
		}
	}

	if a.db != nil {
		if err := a.db.Close(); err != nil {
			logging.Sugar.Errorw("Failed to close database", "error", err)
		}
	}

	logging.Sugar.Infow("Graceful shutdown completed")
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
		if err := repository.RunPostgresMigrations(db); err != nil {
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
