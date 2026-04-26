package repository

import (
	"database/sql"
	"embed"
	"errors"

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
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	defer source.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}
