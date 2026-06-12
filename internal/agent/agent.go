package agent

import (
	"log"
	"time"
)

// generate:reset
type Config struct {
	ServerAddr     string
	PollInterval   time.Duration
	ReportInterval time.Duration
	HashKey        string
	RateLimit      int
}

func DefaultConfig() *Config {
	return &Config{
		ServerAddr:     "localhost:8080",
		PollInterval:   2 * time.Second,
		ReportInterval: 10 * time.Second,
		HashKey:        "",
		RateLimit:      1,
	}
}

type Agent struct {
	config    *Config
	collector *Collector
	pool      *WorkerPool
	stopChan  chan struct{}
}

func New(config *Config) *Agent {
	collector := NewCollector(config.PollInterval)
	pool := NewWorkerPool(config.RateLimit, config.ServerAddr, config.HashKey)

	return &Agent{
		config:    config,
		collector: collector,
		pool:      pool,
		stopChan:  make(chan struct{}),
	}
}

func (a *Agent) Run() {
	a.collector.Start()
	a.pool.Start()

	go a.reportLoop()

	<-a.stopChan
	log.Println("Программная остановка агента...")

	a.cleanup()
}

func (a *Agent) reportLoop() {
	ticker := time.NewTicker(a.config.ReportInterval)
	defer ticker.Stop()

	a.sendMetrics()

	for {
		select {
		case <-a.stopChan:
			return
		case <-ticker.C:
			a.sendMetrics()
		}
	}
}

func (a *Agent) sendMetrics() {
	m := a.collector.GetMetrics()
	metrics := m.ToMetricsSlice()

	log.Printf("Sending %d metrics to worker pool", len(metrics))

	for _, metric := range metrics {
		if !a.pool.Submit(metric) {
			log.Printf("Failed to submit metric %s (pool stopped or queue full)", metric.ID)
		}
	}
}

func (a *Agent) Stop() {
	log.Println("Stopping agent...")

	select {
	case <-a.stopChan:
	default:
		close(a.stopChan)
	}

	a.cleanup()

	log.Println("Agent stopped")
}

func (a *Agent) cleanup() {
	a.collector.Stop()
	a.pool.Stop()
}
