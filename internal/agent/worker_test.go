package agent

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/model"
)

func TestWorkerPool_New(t *testing.T) {
	pool := NewWorkerPool(1, "localhost:8080", "", nil)
	if pool == nil {
		t.Fatal("expected non-nil pool")
	}
	if pool.workers != 1 {
		t.Errorf("expected 1 worker, got %d", pool.workers)
	}
}

func TestWorkerPool_StartStop(t *testing.T) {
	pool := NewWorkerPool(2, "localhost:8080", "", nil)
	pool.Start()

	time.Sleep(10 * time.Millisecond)

	pool.Stop()
}

func TestWorkerPool_Submit(t *testing.T) {
	var receivedCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&receivedCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	addr := server.Listener.Addr().String()

	pool := NewWorkerPool(1, addr, "", nil)
	pool.Start()

	metric := model.Metrics{
		ID:    "test",
		MType: "gauge",
	}

	if !pool.Submit(metric) {
		t.Error("expected Submit to return true")
	}

	time.Sleep(50 * time.Millisecond)

	pool.Stop()

	if atomic.LoadInt32(&receivedCount) == 0 {
		t.Error("expected at least one metric to be received")
	}
}

func TestWorkerPool_SubmitToClosedPool(t *testing.T) {
	pool := NewWorkerPool(1, "localhost:9999", "", nil)
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
	pool := NewWorkerPool(3, "localhost:8080", "", nil)
	if pool.GetWorkersCount() != 3 {
		t.Errorf("expected 3 workers, got %d", pool.GetWorkersCount())
	}
}

func TestWorkerPool_GetServerAddr(t *testing.T) {
	pool := NewWorkerPool(1, "example.com:9090", "", nil)
	if pool.GetServerAddr() != "example.com:9090" {
		t.Errorf("expected 'example.com:9090', got '%s'", pool.GetServerAddr())
	}
}

func TestWorkerPool_GetHashKey(t *testing.T) {
	pool := NewWorkerPool(1, "localhost:8080", "secret", nil)
	if pool.GetHashKey() != "secret" {
		t.Errorf("expected 'secret', got '%s'", pool.GetHashKey())
	}
}

func TestWorkerPool_DefaultWorkers(t *testing.T) {
	pool := NewWorkerPool(0, "localhost:8080", "", nil)
	if pool.GetWorkersCount() != 1 {
		t.Errorf("expected 1 worker for invalid input, got %d", pool.GetWorkersCount())
	}

	pool = NewWorkerPool(-1, "localhost:8080", "", nil)
	if pool.GetWorkersCount() != 1 {
		t.Errorf("expected 1 worker for negative input, got %d", pool.GetWorkersCount())
	}
}
