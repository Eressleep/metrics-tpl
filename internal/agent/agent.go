package agent

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

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

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case <-sigChan:
		log.Println("Получен сигнал завершения, останавливаем агент...")
	case <-a.stopChan:
		log.Println("Программная остановка агента...")
	}

	a.Stop()
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
		a.pool.Submit(metric)
	}
}

func (a *Agent) Stop() {
	log.Println("Stopping agent...")

	a.collector.Stop()
	a.pool.Stop()

	select {
	case <-a.stopChan:
	default:
		close(a.stopChan)
	}

	log.Println("Agent stopped")
}
