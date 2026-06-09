package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/model"
)

func TestWorkerPool_New(t *testing.T) {
	pool := NewWorkerPool(1, "localhost:8080", "")
	if pool == nil {
		t.Fatal("expected non-nil pool")
	}
	if pool.workers != 1 {
		t.Errorf("expected 1 worker, got %d", pool.workers)
	}
}

func TestWorkerPool_StartStop(t *testing.T) {
	pool := NewWorkerPool(2, "localhost:8080", "")
	pool.Start()

	time.Sleep(10 * time.Millisecond)

	pool.Stop()
}

func TestWorkerPool_Submit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	addr := server.Listener.Addr().String()

	pool := NewWorkerPool(1, addr, "")
	pool.Start()
	defer pool.Stop()

	metric := model.Metrics{
		ID:    "test",
		MType: "gauge",
	}

	if !pool.Submit(metric) {
		t.Error("expected Submit to return true")
	}

	time.Sleep(50 * time.Millisecond)
}

func TestWorkerPool_SubmitToFullQueue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	addr := server.Listener.Addr().String()

	pool := &WorkerPool{
		workers:  1,
		jobs:     make(chan Job, 2),
		client:   NewMetricsClient(addr, ""),
		stopChan: make(chan struct{}),
	}
	pool.Start()
	defer pool.Stop()

	metric := model.Metrics{
		ID:    "test",
		MType: "gauge",
	}

	for i := 0; i < 3; i++ {
		pool.Submit(metric)
	}

}

func TestWorkerPool_SubmitToClosedPool(t *testing.T) {
	pool := NewWorkerPool(1, "localhost:9999", "")
	pool.Start()
	pool.Stop()

	metric := model.Metrics{
		ID:    "test",
		MType: "gauge",
	}

	if pool.Submit(metric) {
		t.Error("expected Submit to return false for stopped pool")
	}
}

func TestWorkerPool_GetWorkersCount(t *testing.T) {
	pool := NewWorkerPool(3, "localhost:8080", "")
	if pool.GetWorkersCount() != 3 {
		t.Errorf("expected 3 workers, got %d", pool.GetWorkersCount())
	}
}

func TestWorkerPool_GetServerAddr(t *testing.T) {
	pool := NewWorkerPool(1, "example.com:9090", "")
	if pool.GetServerAddr() != "example.com:9090" {
		t.Errorf("expected 'example.com:9090', got '%s'", pool.GetServerAddr())
	}
}

func TestWorkerPool_GetHashKey(t *testing.T) {
	pool := NewWorkerPool(1, "localhost:8080", "secret")
	if pool.GetHashKey() != "secret" {
		t.Errorf("expected 'secret', got '%s'", pool.GetHashKey())
	}
}

func TestWorkerPool_MultipleSubmits(t *testing.T) {
	receivedCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	addr := server.Listener.Addr().String()

	pool := NewWorkerPool(3, addr, "")
	pool.Start()
	defer pool.Stop()

	metric := model.Metrics{
		ID:    "test",
		MType: "gauge",
	}

	for i := 0; i < 10; i++ {
		pool.Submit(metric)
	}

	time.Sleep(200 * time.Millisecond)

	if receivedCount == 0 {
		t.Error("expected at least one metric to be received")
	}
}
