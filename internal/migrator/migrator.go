package migrator

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/Eressleep/metrics-tpl/pkg/retry"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Migrator struct {
	logger *zap.Logger
}

func NewMigrator(logger *zap.Logger) *Migrator {
	return &Migrator{
		logger: logger,
	}
}

func (m *Migrator) Up(dsn string) error {
	m.logger.Info("Starting database migrations")

	var db *sql.DB

	err := retry.Do(context.TODO(), func() error {
		var err error
		db, err = sql.Open("postgres", dsn)
		if err != nil {
			return err
		}

		if err := db.Ping(); err != nil {
			db.Close()
			return err
		}

		return nil
	}, nil)

	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}

	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	migrator, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
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
