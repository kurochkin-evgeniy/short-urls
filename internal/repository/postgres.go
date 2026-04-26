package repository

import (
	"database/sql"
	"embed"
	"errors"
	"short-urls/internal/logging"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type PostgresKeyValueStorage struct {
	db *sql.DB
}

func NewPostgresKeyValueStorage(db *sql.DB) KeyValueStorage {
	return &PostgresKeyValueStorage{db: db}
}

func (s *PostgresKeyValueStorage) InsertNewValue(key string, value string) (bool, string) {
	const query = `
WITH inserted AS (
    INSERT INTO short_urls (short_url, original_url)
    VALUES ($1, $2)
    ON CONFLICT (original_url) DO NOTHING
    RETURNING short_url
)
SELECT short_url, true AS inserted
FROM inserted
UNION ALL
SELECT short_url, false AS inserted
FROM short_urls
WHERE original_url = $2
  AND NOT EXISTS (SELECT 1 FROM inserted)
LIMIT 1`

	var shortURL string
	var inserted bool
	err := s.db.QueryRow(query, key, value).Scan(&shortURL, &inserted)
	if err != nil {
		return false, ""
	}

	if inserted {
		return true, ""
	}

	return false, shortURL
}

func (s *PostgresKeyValueStorage) GetValue(key string) string {
	const query = `
SELECT original_url
FROM short_urls
WHERE short_url = $1`

	var value string
	if err := s.db.QueryRow(query, key).Scan(&value); err != nil {
		return ""
	}

	return value
}

func runMigrations(db *sql.DB) error {
	logging.Sugar.Debugw("Preparing embedded migration source")
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		logging.Sugar.Errorw("Failed to initialize migration source", "error", err)
		return err
	}
	defer source.Close()

	logging.Sugar.Debugw("Preparing PostgreSQL migration driver")
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		logging.Sugar.Errorw("Failed to initialize PostgreSQL migration driver", "error", err)
		return err
	}

	logging.Sugar.Debugw("Creating migration instance")
	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		logging.Sugar.Errorw("Failed to initialize migration instance", "error", err)
		return err
	}
	defer m.Close()

	logging.Sugar.Debugw("Applying up migrations")
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logging.Sugar.Errorw("Failed to apply migrations", "error", err)
		return err
	} else if errors.Is(err, migrate.ErrNoChange) {
		logging.Sugar.Debugw("No migration changes were needed")
	}

	return nil
}

func RunPostgresMigrations(dsn string) error {
	logging.Sugar.Debugw("Opening dedicated PostgreSQL connection for migrations")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		logging.Sugar.Errorw("Failed to open PostgreSQL connection for migrations", "error", err)
		return err
	}
	defer db.Close()

	logging.Sugar.Debugw("Pinging PostgreSQL on migration connection")
	if err := db.Ping(); err != nil {
		logging.Sugar.Errorw("Failed to ping PostgreSQL on migration connection", "error", err)
		return err
	}

	return runMigrations(db)
}
