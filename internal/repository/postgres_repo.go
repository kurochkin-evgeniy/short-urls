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

func (s *PostgresKeyValueStorage) InsertNewValuesBatch(items []BatchInsertItem) ([]BatchInsertResult, error) {
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
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	results := make([]BatchInsertResult, 0, len(items))
	for _, item := range items {
		var shortURL string
		var inserted bool
		if err := tx.QueryRow(query, item.Key, item.Value).Scan(&shortURL, &inserted); err != nil {
			return nil, err
		}
		result := BatchInsertResult{Inserted: inserted}
		if !inserted {
			result.ExistingKey = shortURL
		}
		results = append(results, result)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return results, nil
}

func (s *PostgresKeyValueStorage) InsertNewValue(key string, value string) (bool, string, error) {
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
		return false, "", err
	}

	if inserted {
		return true, "", nil
	}

	return false, shortURL, nil
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

func RunPostgresMigrations(db *sql.DB) error {
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
