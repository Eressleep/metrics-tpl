package storage

// Storage определяет контракт для работы с хранилищем метрик
type Storage interface {
	UpdateCounter(name string, value int64) error
	UpdateGauge(name string, value float64) error
}
