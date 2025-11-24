package migrations

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

var ErrNilDatabase = errors.New("database handle is nil")

// Run executes all up migrations embedded in this package.
func Run(db *sql.DB) error {
	if db == nil {
		return ErrNilDatabase
	}

	sourceDriver, err := iofs.New(Files, "sql")
	if err != nil {
		return fmt.Errorf("init migration source: %w", err)
	}
	defer sourceDriver.Close()

	dbDriver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("init migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", dbDriver)
	if err != nil {
		return fmt.Errorf("init migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
