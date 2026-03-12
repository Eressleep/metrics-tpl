package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSenderStartStop(t *testing.T) {
	collector := NewCollector(100 * time.Millisecond)
	sender := NewSender("localhost:8080", 100*time.Millisecond, collector)

	sender.Start()
	time.Sleep(50 * time.Millisecond)
	sender.Stop()
}

func TestSenderSendMetric(t *testing.T) {
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

	err := sender.sendMetric("gauge", "test", 123.45)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestSenderSendAllMetrics(t *testing.T) {
	requestCount := 0
	requestChan := make(chan int, 100)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		requestChan <- requestCount
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	collector := NewCollector(50 * time.Millisecond)
	collector.Start()
	defer collector.Stop()

	time.Sleep(100 * time.Millisecond)

	sender := NewSender(server.Listener.Addr().String(), 1*time.Second, collector)
	sender.sendAllMetrics()

	time.Sleep(100 * time.Millisecond)

	if requestCount < 10 {
		t.Errorf("Expected at least 10 requests, got %d", requestCount)
	}
}

func TestSenderSendWithRetry(t *testing.T) {
	attemptCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	collector := NewCollector(1 * time.Second)
	sender := NewSender(server.Listener.Addr().String(), 1*time.Second, collector)
	sender.maxRetries = 3

	done := make(chan bool)
	go func() {
		sender.sendWithRetry("gauge", "test", 123.45)
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out")
	}

	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}
