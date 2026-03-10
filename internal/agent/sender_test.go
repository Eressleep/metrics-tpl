package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSenderSendMetric(t *testing.T) {
	// Создаём тестовый сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "text/plain" {
			t.Errorf("Expected text/plain, got %s", r.Header.Get("Content-Type"))
		}

		expectedPath := "/update/gauge/test/123.45"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	collector := NewCollector(1 * time.Second)
	sender := NewSender(server.Listener.Addr().String(), 1*time.Second, collector)

	sender.sendMetric("gauge", "test", 123.45)
}

func TestSenderSendAllMetrics(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	collector := NewCollector(50 * time.Millisecond)
	collector.Start()
	defer collector.Stop()

	time.Sleep(100 * time.Millisecond)

	sender := NewSender(server.Listener.Addr().String(), 1*time.Second, collector)
	sender.sendAllMetrics()

	if requestCount < 10 {
		t.Errorf("Expected at least 10 requests, got %d", requestCount)
	}
}
