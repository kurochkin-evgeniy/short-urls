package repository

import (
	"database/sql"
	"embed"
	"errors"
	"short-urls/internal/logging"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/lib/pq"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Storage соответствует строке таблицы short_urls в PostgreSQL.
type Storage struct {
	UUID        string `db:"user_id"`
	ShortURL    string `db:"short_url"`
	OriginalURL string `db:"original_url"`
	DeletedFlag bool   `db:"is_deleted"`
}

// PostgresKeyValueStorage реализует KeyValueStorage с использованием PostgreSQL.
type PostgresKeyValueStorage struct {
	db *sql.DB
}

// NewPostgresKeyValueStorage создаёт хранилище на PostgreSQL с заданным подключением.
func NewPostgresKeyValueStorage(db *sql.DB) KeyValueStorage {
	return &PostgresKeyValueStorage{db: db}
}

func (s *PostgresKeyValueStorage) InsertNewValuesBatch(items []BatchInsertItem, userID string) ([]BatchInsertResult, error) {
	const query = `
WITH inserted AS (
    INSERT INTO short_urls (short_url, original_url, user_id)
    VALUES ($1, $2, $3)
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
		if err := tx.QueryRow(query, item.Key, item.Value, userID).Scan(&shortURL, &inserted); err != nil {
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

func (s *PostgresKeyValueStorage) InsertNewValue(key string, value string, userID string) (bool, string, error) {
	const query = `
WITH inserted AS (
    INSERT INTO short_urls (short_url, original_url, user_id)
    VALUES ($1, $2, $3)
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
	err := s.db.QueryRow(query, key, value, userID).Scan(&shortURL, &inserted)
	if err != nil {
		return false, "", err
	}

	if inserted {
		return true, "", nil
	}

	return false, shortURL, nil
}

func (s *PostgresKeyValueStorage) LookupShortURL(key string) (string, bool, bool) {
	const query = `
SELECT original_url, is_deleted
FROM short_urls
WHERE short_url = $1`

	var value string
	var isDeleted bool
	if err := s.db.QueryRow(query, key).Scan(&value, &isDeleted); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, false
		}
		return "", false, false
	}

	return value, isDeleted, true
}

func (s *PostgresKeyValueStorage) MarkURLsDeletedBatch(userID string, shortURLs []string) error {
	if len(shortURLs) == 0 {
		return nil
	}
	const q = `
UPDATE short_urls
SET is_deleted = TRUE
WHERE user_id = $1
  AND short_url = ANY($2::text[])`
	_, err := s.db.Exec(q, userID, pq.Array(shortURLs))
	return err
}

func (s *PostgresKeyValueStorage) GetUserURLs(userID string) ([]UserURL, error) {
	const query = `
SELECT short_url, original_url
FROM short_urls
WHERE user_id = $1
  AND is_deleted = FALSE`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]UserURL, 0)
	for rows.Next() {
		var item UserURL
		if err := rows.Scan(&item.ShortURL, &item.OriginalURL); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *PostgresKeyValueStorage) GetStats() (Stats, error) {
	const query = `
SELECT
    COUNT(*)::int AS urls,
    COUNT(DISTINCT user_id)::int AS users
FROM short_urls`

	var stats Stats
	if err := s.db.QueryRow(query).Scan(&stats.URLs, &stats.Users); err != nil {
		return Stats{}, err
	}
	return stats, nil
}

// RunPostgresMigrations применяет встроенные SQL-миграции к базе данных.
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

	logging.Sugar.Debugw("Applying up migrations")
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logging.Sugar.Errorw("Failed to apply migrations", "error", err)
		return err
	} else if errors.Is(err, migrate.ErrNoChange) {
		logging.Sugar.Debugw("No migration changes were needed")
	}

	return nil
}
