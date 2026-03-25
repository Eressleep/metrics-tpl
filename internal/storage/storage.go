// internal/storage/storage.go
package storage

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/Eressleep/metrics-tpl/internal/database"
)

type Storage interface {
	UpdateCounter(name string, value int64) error
	UpdateGauge(name string, value float64) error
	GetCounter(name string) (int64, error)
	GetGauge(name string) (float64, error)
	GetAllGauges() map[string]float64
	GetAllCounters() map[string]int64
}

type DBStorage struct {
	db *database.DB
	mu sync.RWMutex
}

func NewDBStorage(dsn string) (*DBStorage, error) {
	db, err := database.NewDB(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Создаем таблицы если они не существуют
	if err := initTables(db); err != nil {
		return nil, fmt.Errorf("failed to init tables: %w", err)
	}

	return &DBStorage{
		db: db,
	}, nil
}

func initTables(db *database.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS gauges (
			name TEXT PRIMARY KEY,
			value DOUBLE PRECISION NOT NULL,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS counters (
			name TEXT PRIMARY KEY,
			value BIGINT NOT NULL,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
	}

	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}

func (s *DBStorage) UpdateCounter(name string, value int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `INSERT INTO counters (name, value) VALUES ($1, $2)
			  ON CONFLICT (name) DO UPDATE SET value = counters.value + $2, updated_at = NOW()`
	_, err := s.db.Exec(query, name, value)
	return err
}

func (s *DBStorage) UpdateGauge(name string, value float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `INSERT INTO gauges (name, value) VALUES ($1, $2)
			  ON CONFLICT (name) DO UPDATE SET value = $2, updated_at = NOW()`
	_, err := s.db.Exec(query, name, value)
	return err
}

func (s *DBStorage) GetCounter(name string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var value int64
	err := s.db.QueryRow("SELECT value FROM counters WHERE name = $1", name).Scan(&value)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("counter %s not found", name)
	}
	return value, err
}

func (s *DBStorage) GetGauge(name string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var value float64
	err := s.db.QueryRow("SELECT value FROM gauges WHERE name = $1", name).Scan(&value)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("gauge %s not found", name)
	}
	return value, err
}

func (s *DBStorage) GetAllGauges() map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]float64)
	rows, err := s.db.Query("SELECT name, value FROM gauges")
	if err != nil {
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value float64
		if err := rows.Scan(&name, &value); err == nil {
			result[name] = value
		}
	}
	return result
}

func (s *DBStorage) GetAllCounters() map[string]int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]int64)
	rows, err := s.db.Query("SELECT name, value FROM counters")
	if err != nil {
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value int64
		if err := rows.Scan(&name, &value); err == nil {
			result[name] = value
		}
	}
	return result
}

func (s *DBStorage) Ping() error {
	return s.db.Ping()
}

func (s *DBStorage) Close() error {
	return s.db.Close()
}
