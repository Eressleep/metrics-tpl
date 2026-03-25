package storage

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/migrator"
	"github.com/Eressleep/metrics-tpl/pkg/retry"
	"github.com/jackc/pgerrcode"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

type DBStorage struct {
	db     *sql.DB
	logger *zap.Logger
	mu     sync.RWMutex
}

func NewDBStorage(dsn string, logger *zap.Logger) (*DBStorage, error) {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	m := migrator.NewMigrator(logger)
	if err := retry.Do(context.Background(), func() error {
		return m.Up(dsn)
	}, nil); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	var db *sql.DB
	err := retry.Do(context.Background(), func() error {
		var err error
		db, err = sql.Open("postgres", dsn)
		if err != nil {
			return err
		}

		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(25)
		db.SetConnMaxLifetime(5 * time.Minute)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		return db.PingContext(ctx)
	}, nil)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	logger.Info("Database connection established successfully")

	return &DBStorage{
		db:     db,
		logger: logger,
	}, nil
}

func isRetriablePostgresError(err error) bool {
	if err == nil {
		return false
	}

	if pgErr, ok := err.(*pq.Error); ok {
		connectionErrors := []string{
			pgerrcode.ConnectionException,
			pgerrcode.ConnectionDoesNotExist,
			pgerrcode.ConnectionFailure,
			pgerrcode.SQLClientUnableToEstablishSQLConnection,
			pgerrcode.SQLServerRejectedEstablishmentOfSQLConnection,
			pgerrcode.TransactionResolutionUnknown,
			pgerrcode.ProtocolViolation,
		}

		for _, code := range connectionErrors {
			if string(pgErr.Code) == code {
				return true
			}
		}
	}

	return retry.IsRetriable(err)
}

func (s *DBStorage) execWithRetry(ctx context.Context, query string, args ...interface{}) error {
	return retry.Do(ctx, func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_, err := s.db.ExecContext(ctx, query, args...)
		if err != nil && !isRetriablePostgresError(err) {
			return err
		}
		return err
	}, nil)
}

func (s *DBStorage) UpdateCounter(name string, value int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO counters (name, value) 
		VALUES ($1, $2)
		ON CONFLICT (name) 
		DO UPDATE SET value = counters.value + $2
	`

	return s.execWithRetry(ctx, query, name, value)
}

func (s *DBStorage) UpdateGauge(name string, value float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO gauges (name, value) 
		VALUES ($1, $2)
		ON CONFLICT (name) 
		DO UPDATE SET value = $2
	`

	return s.execWithRetry(ctx, query, name, value)
}

func (s *DBStorage) BatchUpdate(ctx context.Context, metrics []Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return retry.Do(ctx, func() error {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		gaugeQuery := `
			INSERT INTO gauges (name, value) 
			VALUES ($1, $2)
			ON CONFLICT (name) 
			DO UPDATE SET value = $2
		`

		counterQuery := `
			INSERT INTO counters (name, value) 
			VALUES ($1, $2)
			ON CONFLICT (name) 
			DO UPDATE SET value = counters.value + $2
		`

		for _, m := range metrics {
			switch m.MType {
			case "counter":
				if m.Delta != nil {
					if _, err := tx.ExecContext(ctx, counterQuery, m.ID, *m.Delta); err != nil {
						return err
					}
				}
			case "gauge":
				if m.Value != nil {
					if _, err := tx.ExecContext(ctx, gaugeQuery, m.ID, *m.Value); err != nil {
						return err
					}
				}
			}
		}

		return tx.Commit()
	}, nil)
}

func (s *DBStorage) GetCounter(name string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var value int64

	err := retry.Do(ctx, func() error {
		row := s.db.QueryRowContext(ctx, "SELECT value FROM counters WHERE name = $1", name)
		err := row.Scan(&value)

		if err == sql.ErrNoRows {
			return fmt.Errorf("counter %s not found", name)
		}

		if err != nil && !isRetriablePostgresError(err) {
			return err
		}

		return err
	}, nil)

	if err != nil {
		return 0, fmt.Errorf("failed to get counter %s: %w", name, err)
	}

	return value, nil
}

func (s *DBStorage) GetGauge(name string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var value float64

	err := retry.Do(ctx, func() error {
		row := s.db.QueryRowContext(ctx, "SELECT value FROM gauges WHERE name = $1", name)
		err := row.Scan(&value)

		if err == sql.ErrNoRows {
			return fmt.Errorf("gauge %s not found", name)
		}

		if err != nil && !isRetriablePostgresError(err) {
			return err
		}

		return err
	}, nil)

	if err != nil {
		return 0, fmt.Errorf("failed to get gauge %s: %w", name, err)
	}

	return value, nil
}

func (s *DBStorage) GetAllGauges() map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]float64)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var rows *sql.Rows
	err := retry.Do(ctx, func() error {
		var err error
		rows, err = s.db.QueryContext(ctx, "SELECT name, value FROM gauges ORDER BY name")
		if err != nil && !isRetriablePostgresError(err) {
			return err
		}
		return err
	}, nil)

	if err != nil {
		s.logger.Error("Failed to get all gauges", zap.Error(err))
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value float64
		if err := rows.Scan(&name, &value); err != nil {
			s.logger.Error("Failed to scan gauge", zap.Error(err))
			continue
		}
		result[name] = value
	}

	if err := rows.Err(); err != nil {
		s.logger.Error("Error iterating gauges", zap.Error(err))
	}

	return result
}

func (s *DBStorage) GetAllCounters() map[string]int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]int64)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var rows *sql.Rows
	err := retry.Do(ctx, func() error {
		var err error
		rows, err = s.db.QueryContext(ctx, "SELECT name, value FROM counters ORDER BY name")
		if err != nil && !isRetriablePostgresError(err) {
			return err
		}
		return err
	}, nil)

	if err != nil {
		s.logger.Error("Failed to get all counters", zap.Error(err))
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value int64
		if err := rows.Scan(&name, &value); err != nil {
			s.logger.Error("Failed to scan counter", zap.Error(err))
			continue
		}
		result[name] = value
	}

	if err := rows.Err(); err != nil {
		s.logger.Error("Error iterating counters", zap.Error(err))
	}

	return result
}

func (s *DBStorage) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return retry.Do(ctx, func() error {
		return s.db.PingContext(ctx)
	}, nil)
}

func (s *DBStorage) Close() error {
	s.logger.Info("Closing database connection")
	return s.db.Close()
}
