package agent

import (
	"testing"
	"time"
)

func TestCollector(t *testing.T) {
	collector := NewCollector(100 * time.Millisecond)
	collector.Start()

	time.Sleep(150 * time.Millisecond)

	metrics := collector.GetMetrics()

	if metrics.GetPollCount() == 0 {
		t.Error("PollCount should be > 0")
	}

	gauges := metrics.GetAllGauges()
	if len(gauges) == 0 {
		t.Error("Gauges should not be empty")
	}

	if _, ok := gauges["RandomValue"]; !ok {
		t.Error("RandomValue metric missing")
	}

	collector.Stop()
}

func TestCollectorMultiplePolls(t *testing.T) {
	collector := NewCollector(50 * time.Millisecond)
	collector.Start()

	time.Sleep(120 * time.Millisecond)

	pollCount1 := collector.GetMetrics().GetPollCount()

	time.Sleep(100 * time.Millisecond)

	pollCount2 := collector.GetMetrics().GetPollCount()

	if pollCount2 <= pollCount1 {
		t.Errorf("PollCount should increase: before=%d, after=%d", pollCount1, pollCount2)
	}

	collector.Stop()
}

func TestCollectorStop(t *testing.T) {
	collector := NewCollector(10 * time.Millisecond)
	collector.Start()

	time.Sleep(50 * time.Millisecond)

	pollCountBefore := collector.GetMetrics().GetPollCount()
	t.Logf("PollCount before stop: %d", pollCountBefore)

	collector.Stop()

	time.Sleep(30 * time.Millisecond)

	pollCountAfter := collector.GetMetrics().GetPollCount()
	t.Logf("PollCount after stop: %d", pollCountAfter)

	if pollCountAfter > pollCountBefore+1 {
		t.Errorf("PollCount increased too much after stop: before=%d, after=%d",
			pollCountBefore, pollCountAfter)
	}
}

func TestCollectorConcurrency(t *testing.T) {
	collector := NewCollector(10 * time.Millisecond)
	collector.Start()
	defer collector.Stop()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				collector.GetMetrics().GetAllGauges()
				collector.GetMetrics().GetPollCount()
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestAgentNew(t *testing.T) {
	config := &Config{
		ServerAddr:     "test:8080",
		PollInterval:   1 * time.Second,
		ReportInterval: 2 * time.Second,
		CryptoKeyPath:  "",
	}

	agt := New(config)
	if agt == nil {
		t.Fatal("Agent should not be nil")
	}

	if agt.config.ServerAddr != "test:8080" {
		t.Errorf("Expected ServerAddr test:8080, got %s", agt.config.ServerAddr)
	}
}

func TestAgentDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config.ServerAddr != "localhost:8080" {
		t.Errorf("Expected default ServerAddr localhost:8080, got %s", config.ServerAddr)
	}
}

func TestWorkerPoolStop(t *testing.T) {
	pool := NewWorkerPool(2, "localhost:8080", "", nil)
	pool.Start()

	if pool.GetServerAddr() != "localhost:8080" {
		t.Errorf("Expected server addr 'localhost:8080', got '%s'", pool.GetServerAddr())
	}

	pool.Stop()
}

func TestWorkerPoolDoubleStop(t *testing.T) {
	pool := NewWorkerPool(2, "localhost:8080", "", nil)
	pool.Start()

	pool.Stop()
	pool.Stop()

	t.Log("Double Stop completed without panic")
}

func TestCollectorStartStop(t *testing.T) {
	collector := NewCollector(100 * time.Millisecond)

	for i := 0; i < 3; i++ {
		collector.Start()
		time.Sleep(50 * time.Millisecond)
		collector.Stop()
	}
}

func BenchmarkCollector(b *testing.B) {
	collector := NewCollector(time.Microsecond)
	collector.Start()
	defer collector.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.GetMetrics().GetAllGauges()
		collector.GetMetrics().GetPollCount()
	}
}
