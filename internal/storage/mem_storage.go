package storage

import (
	"context"
	"errors"
	"sync"
)

type MemStorage struct {
	gauges   map[string]float64
	counters map[string]int64
	mu       sync.RWMutex
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64, 32),
		counters: make(map[string]int64, 8),
	}
}

func (s *MemStorage) UpdateCounter(name string, value int64) error {
	if name == "" {
		return errors.New("metric name cannot be empty")
	}

	s.mu.Lock()
	s.counters[name] += value
	s.mu.Unlock()
	return nil
}

func (s *MemStorage) UpdateGauge(name string, value float64) error {
	if name == "" {
		return errors.New("metric name cannot be empty")
	}

	s.mu.Lock()
	s.gauges[name] = value
	s.mu.Unlock()
	return nil
}

func (s *MemStorage) BatchUpdate(ctx context.Context, metrics []Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	for i := range metrics {
		m := &metrics[i]
		switch m.MType {
		case "counter":
			if m.Delta != nil {
				s.counters[m.ID] += *m.Delta
			}
		case "gauge":
			if m.Value != nil {
				s.gauges[m.ID] = *m.Value
			}
		}
	}

	return nil
}

func (s *MemStorage) GetCounter(name string) (int64, error) {
	s.mu.RLock()
	val, ok := s.counters[name]
	s.mu.RUnlock()

	if !ok {
		return 0, errors.New("counter not found")
	}
	return val, nil
}

func (s *MemStorage) GetGauge(name string) (float64, error) {
	s.mu.RLock()
	val, ok := s.gauges[name]
	s.mu.RUnlock()

	if !ok {
		return 0, errors.New("gauge not found")
	}
	return val, nil
}

func (s *MemStorage) GetAllGauges() map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent concurrent access issues
	gauges := make(map[string]float64, len(s.gauges))
	for k, v := range s.gauges {
		gauges[k] = v
	}
	return gauges
}

func (s *MemStorage) GetAllCounters() map[string]int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	counters := make(map[string]int64, len(s.counters))
	for k, v := range s.counters {
		counters[k] = v
	}
	return counters
}
