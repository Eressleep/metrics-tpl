package agent

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
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

func TestNewSenderWithPublicKey(t *testing.T) {
	collector := NewCollector(time.Second)

	sender := NewSender("localhost:8080", time.Second, collector, "", nil)
	if sender == nil {
		t.Fatal("Sender should not be nil")
	}
	if sender.client.publicKey != nil {
		t.Error("Public key should be nil when not provided")
	}

	sender = NewSender("localhost:8080", time.Second, collector, "", nil)
	if sender.client.publicKey != nil {
		t.Error("Public key should be nil when key path is empty")
	}
}
