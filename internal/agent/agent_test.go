package agent

import (
	"testing"
	"time"
)

func TestCollector(t *testing.T) {
	collector := NewCollector(100 * time.Millisecond)
	collector.Start()

	// Даём время на первый сбор
	time.Sleep(150 * time.Millisecond)

	metrics := collector.GetMetrics()

	// Проверяем, что метрики собрались
	if metrics.GetPollCount() == 0 {
		t.Error("PollCount should be > 0")
	}

	gauges := metrics.GetAllGauges()
	if len(gauges) == 0 {
		t.Error("Gauges should not be empty")
	}

	// Проверяем наличие RandomValue
	if _, ok := gauges["RandomValue"]; !ok {
		t.Error("RandomValue metric missing")
	}

	collector.Stop()
}

func TestCollectorMultiplePolls(t *testing.T) {
	collector := NewCollector(50 * time.Millisecond)
	collector.Start()

	// Ждём несколько интервалов
	time.Sleep(120 * time.Millisecond)

	pollCount1 := collector.GetMetrics().GetPollCount()

	time.Sleep(100 * time.Millisecond)

	pollCount2 := collector.GetMetrics().GetPollCount()

	if pollCount2 <= pollCount1 {
		t.Errorf("PollCount should increase: before=%d, after=%d", pollCount1, pollCount2)
	}

	collector.Stop()
}

type MockSender struct {
	sentMetrics map[string]interface{}
}

func NewMockSender() *MockSender {
	return &MockSender{
		sentMetrics: make(map[string]interface{}),
	}
}

func TestSender(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	collector := NewCollector(50 * time.Millisecond)
	collector.Start()
	defer collector.Stop()

	sender := NewSender(server.Addr(), 100*time.Millisecond, collector)
	sender.Start()
	defer sender.Stop()

	// Ждём отправки
	time.Sleep(250 * time.Millisecond)

	// Проверяем, что сервер получил метрики
	if server.GetRequestCount() == 0 {
		t.Error("Server should have received requests")
	}
}

func NewTestServer() *TestServer {
	return &TestServer{
		requestCount: 0,
		addr:         "localhost:8081",
	}
}

type TestServer struct {
	requestCount int
	addr         string
}

func (s *TestServer) Close() {}

func (s *TestServer) Addr() string { return s.addr }

func (s *TestServer) GetRequestCount() int { return s.requestCount }
