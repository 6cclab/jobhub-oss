// Package db provides the Postgres connection and migration bootstrap for
// the job search dashboard server. It uses jackc/pgx/v5 (via its
// database/sql stdlib adapter) and golang-migrate/migrate for schema
// migrations.
package db

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Open connects to a Postgres database using the provided DSN.
// Example: postgres://jobhub:pass@localhost:5432/jobhub?sslmode=disable
func Open(databaseURL string) (*sql.DB, error) {
	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}

	sqlDB.SetMaxOpenConns(10)

	return sqlDB, nil
}

// Migrate applies all pending "up" migrations from migrationsFS.
func Migrate(db *sql.DB, migrationsFS embed.FS) error {
	sourceDriver, err := iofs.New(migrationsFS, ".")
	if err != nil {
		return fmt.Errorf("db: create migration source: %w", err)
	}

	dbDriver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("db: create migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", dbDriver)
	if err != nil {
		return fmt.Errorf("db: create migrator: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("db: apply migrations: %w", err)
	}

	return nil
}
