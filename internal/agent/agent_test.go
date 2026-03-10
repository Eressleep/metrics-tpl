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

	collector.Stop()

	pollCount1 := collector.GetMetrics().GetPollCount()
	time.Sleep(100 * time.Millisecond)
	pollCount2 := collector.GetMetrics().GetPollCount()

	if pollCount2 != pollCount1 {
		t.Errorf("PollCount should not increase after stop: before=%d, after=%d",
			pollCount1, pollCount2)
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

func TestCollectorAndSenderIntegration(t *testing.T) {
	collector := NewCollector(50 * time.Millisecond)
	collector.Start()
	defer collector.Stop()

	time.Sleep(100 * time.Millisecond)

	metrics := collector.GetMetrics()
	if metrics.GetPollCount() == 0 {
		t.Error("Collector should collect metrics")
	}

	gauges := metrics.GetAllGauges()
	if len(gauges) == 0 {
		t.Error("Gauges should not be empty")
	}
}
