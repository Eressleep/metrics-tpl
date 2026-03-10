package agent

import (
	"net/http"
	"net/http/httptest"
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
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	collector := NewCollector(50 * time.Millisecond)
	collector.Start()
	defer collector.Stop()

	sender := NewSender(server.Listener.Addr().String(), 100*time.Millisecond, collector)
	sender.Start()
	defer sender.Stop()

	time.Sleep(250 * time.Millisecond)

	if requestCount == 0 {
		t.Error("Server should have received requests")
	}
}

type TestServer struct {
	server       *httptest.Server
	requestCount int
}

func NewTestServer() *TestServer {
	ts := &TestServer{
		requestCount: 0,
	}

	ts.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts.requestCount++
		w.WriteHeader(http.StatusOK)
	}))

	return ts
}

func (s *TestServer) Close() {
	if s.server != nil {
		s.server.Close()
	}
}

func (s *TestServer) Addr() string {
	return s.server.Listener.Addr().String()
}

func (s *TestServer) GetRequestCount() int {
	return s.requestCount
}
