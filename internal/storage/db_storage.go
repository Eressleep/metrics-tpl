package storage

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/migrator"
	_ "github.com/lib/pq"
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

	// Выполняем миграции
	m := migrator.NewMigrator(logger)
	if err := m.Up(dsn); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Подключаемся к БД
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Настройка пула соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Проверяем соединение
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database connection established successfully")

	return &DBStorage{
		db:     db,
		logger: logger,
	}, nil
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

	_, err := s.db.ExecContext(ctx, query, name, value)
	if err != nil {
		s.logger.Error("Failed to update counter",
			zap.String("name", name),
			zap.Int64("delta", value),
			zap.Error(err))
		return fmt.Errorf("failed to update counter %s: %w", name, err)
	}

	return nil
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

	_, err := s.db.ExecContext(ctx, query, name, value)
	if err != nil {
		s.logger.Error("Failed to update gauge",
			zap.String("name", name),
			zap.Float64("value", value),
			zap.Error(err))
		return fmt.Errorf("failed to update gauge %s: %w", name, err)
	}

	return nil
}

func (s *DBStorage) GetCounter(name string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var value int64
	err := s.db.QueryRowContext(ctx, "SELECT value FROM counters WHERE name = $1", name).Scan(&value)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("counter %s not found", name)
	}
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
	err := s.db.QueryRowContext(ctx, "SELECT value FROM gauges WHERE name = $1", name).Scan(&value)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("gauge %s not found", name)
	}
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

	rows, err := s.db.QueryContext(ctx, "SELECT name, value FROM gauges ORDER BY name")
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

	// Проверяем ошибки после итерации
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

	rows, err := s.db.QueryContext(ctx, "SELECT name, value FROM counters ORDER BY name")
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

	// Проверяем ошибки после итерации
	if err := rows.Err(); err != nil {
		s.logger.Error("Error iterating counters", zap.Error(err))
	}

	return result
}

func (s *DBStorage) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return s.db.PingContext(ctx)
}

func (s *DBStorage) Close() error {
	s.logger.Info("Closing database connection")
	return s.db.Close()
}
