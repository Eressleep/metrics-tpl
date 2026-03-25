package migrator

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

//go:embed ../database/migrations/*.sql
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
	m.logger.Info("Starting database migrations", zap.String("dsn", maskDSN(dsn)))

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}

	source, err := iofs.New(migrationsFS, "../database/migrations")
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

	m.logger.Info("Database migrations completed successfully")
	return nil
}

func (m *Migrator) Down(dsn string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}

	source, err := iofs.New(migrationsFS, "../database/migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	migrator, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	if err := migrator.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}

	return nil
}

func maskDSN(dsn string) string {
	return "postgres://****:****@****/****"
}

func (m *Migrator) Version(dsn string) (uint, bool, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return 0, false, err
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return 0, false, err
	}

	source, err := iofs.New(migrationsFS, "../database/migrations")
	if err != nil {
		return 0, false, err
	}

	migrator, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return 0, false, err
	}

	version, dirty, err := migrator.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return 0, false, err
	}

	return version, dirty, nil
}
