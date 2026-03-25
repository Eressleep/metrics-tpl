package metrics

import (
	"testing"
)

func TestMetricsUpdateRuntime(t *testing.T) {
	m := New()

	m.UpdateRuntime()

	gauges := m.GetAllGauges()

	expectedMetrics := []string{
		"Alloc", "HeapAlloc", "Sys", "TotalAlloc", "NumGC",
	}

	for _, name := range expectedMetrics {
		if _, ok := gauges[name]; !ok {
			t.Errorf("Metric %s not found", name)
		}
	}
}

func TestMetricsUpdateRandom(t *testing.T) {
	m := New()

	m.UpdateRandom()
	val1 := m.RandomValue

	m.UpdateRandom()
	val2 := m.RandomValue

	if val1 == val2 {
		t.Log("Warning: Random values are equal, but this is possible")
	}
}

func TestMetricsIncrementPollCount(t *testing.T) {
	m := New()

	if m.GetPollCount() != 0 {
		t.Errorf("Expected PollCount=0, got %d", m.GetPollCount())
	}

	m.IncrementPollCount()
	if m.GetPollCount() != 1 {
		t.Errorf("Expected PollCount=1, got %d", m.GetPollCount())
	}

	m.IncrementPollCount()
	if m.GetPollCount() != 2 {
		t.Errorf("Expected PollCount=2, got %d", m.GetPollCount())
	}
}

func TestMetricsConcurrency(t *testing.T) {
	m := New()

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				m.UpdateRuntime()
				m.UpdateRandom()
				m.IncrementPollCount()
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	gauges := m.GetAllGauges()
	if len(gauges) == 0 {
		t.Error("Gauges should not be empty")
	}

	if m.GetPollCount() != 1000 {
		t.Errorf("Expected PollCount=1000, got %d", m.GetPollCount())
	}
}
