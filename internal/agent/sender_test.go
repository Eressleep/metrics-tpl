package agent

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/model"
	"github.com/mailru/easyjson"
)

func TestSenderJSONStartStop(t *testing.T) {
	collector := NewCollector(100 * time.Millisecond)
	sender := NewSender("localhost:8080", 100*time.Millisecond, collector)

	sender.Start()
	time.Sleep(50 * time.Millisecond)
	sender.Stop()
}

func TestSenderSendMetricJSON(t *testing.T) {
	var receivedMetric model.Metrics

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/update" {
			t.Errorf("Expected path /update, got %s", r.URL.Path)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected application/json, got %s", r.Header.Get("Content-Type"))
		}

		var bodyReader io.Reader = r.Body
		if r.Header.Get("Content-Encoding") == "gzip" {
			gzipReader, err := gzip.NewReader(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			defer gzipReader.Close()
			bodyReader = gzipReader
		}

		body, err := io.ReadAll(bodyReader)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()

		err = easyjson.Unmarshal(body, &receivedMetric)
		if err != nil {
			t.Errorf("Error unmarshaling JSON: %v", err)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	collector := NewCollector(1 * time.Second)
	sender := NewSender(server.Listener.Addr().String(), 1*time.Second, collector)
	sender.useGzip = false // Отключаем gzip для этого теста

	value := 123.45
	metric := model.Metrics{
		ID:    "test",
		MType: model.Gauge,
		Value: &value,
	}

	err := sender.sendMetricJSON(metric)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if receivedMetric.ID != "test" {
		t.Errorf("Expected ID 'test', got '%s'", receivedMetric.ID)
	}
	if receivedMetric.MType != model.Gauge {
		t.Errorf("Expected MType 'gauge', got '%s'", receivedMetric.MType)
	}
	if receivedMetric.Value == nil || *receivedMetric.Value != 123.45 {
		t.Errorf("Expected Value 123.45, got %v", receivedMetric.Value)
	}
}

func TestSenderSendAllMetricsJSON(t *testing.T) {
	requestCount := 0
	receivedMetrics := make(map[string]bool)
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var bodyReader io.Reader = r.Body
		if r.Header.Get("Content-Encoding") == "gzip" {
			gzipReader, err := gzip.NewReader(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			defer gzipReader.Close()
			bodyReader = gzipReader
		}

		body, err := io.ReadAll(bodyReader)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()

		var metric model.Metrics
		err = easyjson.Unmarshal(body, &metric)
		if err != nil {
			t.Errorf("Error unmarshaling JSON: %v", err)
			return
		}

		mu.Lock()
		receivedMetrics[metric.ID] = true
		requestCount++
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	collector := NewCollector(50 * time.Millisecond)
	collector.Start()
	defer collector.Stop()

	time.Sleep(100 * time.Millisecond)

	sender := NewSender(server.Listener.Addr().String(), 1*time.Second, collector)
	sender.useGzip = false // Отключаем gzip для теста
	sender.sendAllMetrics()

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if requestCount < 28 {
		t.Errorf("Expected at least 28 requests, got %d", requestCount)
	}

	expectedMetrics := []string{"Alloc", "HeapAlloc", "Sys", "PollCount"}
	for _, name := range expectedMetrics {
		if !receivedMetrics[name] {
			t.Errorf("Metric %s was not received", name)
		}
	}
}

func TestSenderSendCounterJSON(t *testing.T) {
	var receivedMetric model.Metrics
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var bodyReader io.Reader = r.Body
		if r.Header.Get("Content-Encoding") == "gzip" {
			gzipReader, err := gzip.NewReader(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			defer gzipReader.Close()
			bodyReader = gzipReader
		}

		body, err := io.ReadAll(bodyReader)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()

		var metric model.Metrics
		err = easyjson.Unmarshal(body, &metric)
		if err != nil {
			t.Errorf("Error unmarshaling JSON: %v", err)
			return
		}

		mu.Lock()
		receivedMetric = metric
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	collector := NewCollector(1 * time.Second)
	sender := NewSender(server.Listener.Addr().String(), 1*time.Second, collector)
	sender.useGzip = false // Отключаем gzip для теста

	pollCount := int64(42)
	metric := model.Metrics{
		ID:    "PollCount",
		MType: model.Counter,
		Delta: &pollCount,
	}

	err := sender.sendMetricJSON(metric)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if receivedMetric.ID != "PollCount" {
		t.Errorf("Expected ID 'PollCount', got '%s'", receivedMetric.ID)
	}
	if receivedMetric.MType != model.Counter {
		t.Errorf("Expected MType 'counter', got '%s'", receivedMetric.MType)
	}
	if receivedMetric.Delta == nil || *receivedMetric.Delta != 42 {
		t.Errorf("Expected Delta 42, got %v", receivedMetric.Delta)
	}
}

func TestSenderSendWithRetryJSON(t *testing.T) {
	attemptCount := 0
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attemptCount++
		currentAttempt := attemptCount
		mu.Unlock()

		if currentAttempt < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	collector := NewCollector(1 * time.Second)
	sender := NewSender(server.Listener.Addr().String(), 1*time.Second, collector)
	sender.maxRetries = 3
	sender.useGzip = false // Отключаем gzip для теста

	value := 123.45
	metric := model.Metrics{
		ID:    "test",
		MType: model.Gauge,
		Value: &value,
	}

	done := make(chan bool)
	go func() {
		sender.sendMetricWithRetry(metric)
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out")
	}

	mu.Lock()
	defer mu.Unlock()

	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

func TestSenderSendMetricJSONWithGzip(t *testing.T) {
	var receivedMetric model.Metrics
	var receivedContentEncoding string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentEncoding = r.Header.Get("Content-Encoding")

		var bodyReader io.Reader = r.Body
		if r.Header.Get("Content-Encoding") == "gzip" {
			gzipReader, err := gzip.NewReader(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			defer gzipReader.Close()
			bodyReader = gzipReader
		}

		body, err := io.ReadAll(bodyReader)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()

		err = easyjson.Unmarshal(body, &receivedMetric)
		if err != nil {
			t.Errorf("Error unmarshaling JSON: %v", err)
		}

		if r.Header.Get("Accept-Encoding") != "gzip" {
			t.Error("Expected Accept-Encoding: gzip")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	collector := NewCollector(1 * time.Second)
	sender := NewSender(server.Listener.Addr().String(), 1*time.Second, collector)
	sender.useGzip = true

	value := 123.45
	metric := model.Metrics{
		ID:    "test",
		MType: model.Gauge,
		Value: &value,
	}

	err := sender.sendMetricJSON(metric)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if receivedContentEncoding != "gzip" {
		t.Errorf("Expected Content-Encoding: gzip, got %q", receivedContentEncoding)
	}

	if receivedMetric.ID != "test" {
		t.Errorf("Expected ID 'test', got '%s'", receivedMetric.ID)
	}
	if receivedMetric.MType != model.Gauge {
		t.Errorf("Expected MType 'gauge', got '%s'", receivedMetric.MType)
	}
	if receivedMetric.Value == nil || *receivedMetric.Value != 123.45 {
		t.Errorf("Expected Value 123.45, got %v", receivedMetric.Value)
	}
}
