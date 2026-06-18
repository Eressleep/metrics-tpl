// Package migrator отвечает за миграции базы данных.
package migrator

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"go.uber.org/zap"
)

// Migrator выполняет миграции базы данных
type Migrator struct {
	logger *zap.Logger
}

// NewMigrator создает новый экземпляр Migrator
func NewMigrator(logger *zap.Logger) *Migrator {
	return &Migrator{
		logger: logger,
	}
}

// Up запускает миграции
func (m *Migrator) Up(dsn string) error {
	m.logger.Info("Starting database migrations")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}

	migrator, err := migrate.NewWithDatabaseInstance(
		"file://internal/migrator/migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	if err := migrator.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	version, dirty, _ := migrator.Version()
	m.logger.Info("Database migrations completed",
		zap.Uint("version", version),
		zap.Bool("dirty", dirty))

	return nil
}
