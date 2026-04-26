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

func NewPostgresKeyValueStorage(db *sql.DB) (KeyValueStorage, error) {
	if err := runMigrations(db); err != nil {
		return nil, err
	}

	return &PostgresKeyValueStorage{db: db}, nil
}

func (s *PostgresKeyValueStorage) InsertNewValue(key string, value string) bool {
	const query = `
INSERT INTO short_urls (short_url, original_url)
VALUES ($1, $2)
ON CONFLICT (short_url) DO NOTHING`

	result, err := s.db.Exec(query, key, value)
	if err != nil {
		return false
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false
	}

	return affected > 0
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
