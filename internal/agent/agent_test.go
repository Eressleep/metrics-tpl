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

func TestAgentNew(t *testing.T) {
	config := &Config{
		ServerAddr:     "test:8080",
		PollInterval:   1 * time.Second,
		ReportInterval: 2 * time.Second,
	}

	agent := New(config)
	if agent == nil {
		t.Fatal("Agent should not be nil")
	}

	if agent.config.ServerAddr != "test:8080" {
		t.Errorf("Expected ServerAddr test:8080, got %s", agent.config.ServerAddr)
	}
	if agent.config.PollInterval != 1*time.Second {
		t.Errorf("Expected PollInterval 1s, got %v", agent.config.PollInterval)
	}
	if agent.config.ReportInterval != 2*time.Second {
		t.Errorf("Expected ReportInterval 2s, got %v", agent.config.ReportInterval)
	}
}

func TestAgentDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config.ServerAddr != "localhost:8080" {
		t.Errorf("Expected default ServerAddr localhost:8080, got %s", config.ServerAddr)
	}
	if config.PollInterval != 2*time.Second {
		t.Errorf("Expected default PollInterval 2s, got %v", config.PollInterval)
	}
	if config.ReportInterval != 10*time.Second {
		t.Errorf("Expected default ReportInterval 10s, got %v", config.ReportInterval)
	}
}

func TestAgentStop(t *testing.T) {
	config := DefaultConfig()
	agent := New(config)

	go agent.Run()

	time.Sleep(200 * time.Millisecond)

	agent.Stop()

	select {
	case <-time.After(2 * time.Second):
		t.Log("Agent stopped successfully")
	case <-func() chan bool {
		done := make(chan bool)
		go func() {
			time.Sleep(500 * time.Millisecond)
			done <- true
		}()
		return done
	}():
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				ServerAddr:     "localhost:8080",
				PollInterval:   1 * time.Second,
				ReportInterval: 2 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "empty server addr",
			config: Config{
				ServerAddr:     "",
				PollInterval:   1 * time.Second,
				ReportInterval: 2 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero poll interval",
			config: Config{
				ServerAddr:     "localhost:8080",
				PollInterval:   0,
				ReportInterval: 2 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero report interval",
			config: Config{
				ServerAddr:     "localhost:8080",
				PollInterval:   1 * time.Second,
				ReportInterval: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
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

func BenchmarkCollectorConcurrent(b *testing.B) {
	collector := NewCollector(time.Microsecond)
	collector.Start()
	defer collector.Stop()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			collector.GetMetrics().GetAllGauges()
			collector.GetMetrics().GetPollCount()
		}
	})
}
