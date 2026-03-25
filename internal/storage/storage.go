package storage

type Storage interface {
	UpdateCounter(name string, value int64) error
	UpdateGauge(name string, value float64) error
	GetCounter(name string) (int64, error)
	GetGauge(name string) (float64, error)
	GetAllGauges() map[string]float64
	GetAllCounters() map[string]int64
}
