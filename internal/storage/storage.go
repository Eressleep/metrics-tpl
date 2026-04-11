package storage

import "context"

type Storage interface {
	UpdateCounter(name string, value int64) error
	UpdateGauge(name string, value float64) error
	GetCounter(name string) (int64, error)
	GetGauge(name string) (float64, error)
	GetAllGauges() map[string]float64
	GetAllCounters() map[string]int64

	BatchUpdate(ctx context.Context, metrics []Metrics) error
}

type Metrics struct {
	ID    string
	MType string
	Delta *int64
	Value *float64
}
