// Package model содержит определения структур данных для метрик.
package model

//go:generate easyjson -all metrics.go

const (
	Counter = "counter"
	Gauge   = "gauge"
)

// generate:reset
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

// Reset сбрасывает состояние метрики к начальным значениям
func (m *Metrics) Reset() {
	m.ID = ""
	m.MType = ""
	m.Delta = nil
	m.Value = nil
	m.Hash = ""
}

type BatchMetrics []Metrics

// Reset сбрасывает состояние пакета метрик
func (bm *BatchMetrics) Reset() {
	*bm = (*bm)[:0]
}
