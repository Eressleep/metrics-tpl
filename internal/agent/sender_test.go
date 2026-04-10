package agent

import (
	"sync"
	"sync/atomic"
)

type MockCollector struct {
	gauges              map[string]float64
	pollCount           int64
	mu                  sync.RWMutex
	getMetricsCallCount int32
}

func NewMockCollector() *MockCollector {
	return &MockCollector{
		gauges:    make(map[string]float64),
		pollCount: 0,
	}
}

func (m *MockCollector) SetGauge(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
}

func (m *MockCollector) SetPollCount(count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pollCount = count
}

func (m *MockCollector) GetGetMetricsCallCount() int32 {
	return atomic.LoadInt32(&m.getMetricsCallCount)
}
