package agent

import (
	"crypto/rsa"
	"log"
	"sync"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/crypto"
)

// generate:reset
type Config struct {
	ServerAddr     string
	PollInterval   time.Duration
	ReportInterval time.Duration
	HashKey        string
	RateLimit      int
	CryptoKeyPath  string
}

func DefaultConfig() *Config {
	return &Config{
		ServerAddr:     "localhost:8080",
		PollInterval:   2 * time.Second,
		ReportInterval: 10 * time.Second,
		HashKey:        "",
		RateLimit:      1,
		CryptoKeyPath:  "",
	}
}

type Agent struct {
	config    *Config
	collector *Collector
	pool      *WorkerPool
	stopChan  chan struct{}
	wg        sync.WaitGroup
	mu        sync.Mutex
	stopped   bool
}

func New(config *Config) *Agent {
	collector := NewCollector(config.PollInterval)

	var publicKey *rsa.PublicKey
	if config.CryptoKeyPath != "" {
		var err error
		publicKey, err = crypto.LoadPublicKey(config.CryptoKeyPath)
		if err != nil {
			log.Printf("WARNING: Failed to load public key: %v", err)
		} else {
			log.Printf("Loaded public key from: %s", config.CryptoKeyPath)
		}
	}

	pool := NewWorkerPool(config.RateLimit, config.ServerAddr, config.HashKey, publicKey)

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

	a.wg.Add(1)
	go a.reportLoop()

	<-a.stopChan
	log.Println("Received shutdown signal, starting graceful shutdown...")

	a.gracefulStop()
}

func (a *Agent) reportLoop() {
	defer a.wg.Done()

	ticker := time.NewTicker(a.config.ReportInterval)
	defer ticker.Stop()

	a.sendMetrics()

	for {
		select {
		case <-a.stopChan:
			log.Println("Report loop received stop signal, sending final metrics...")
			a.sendMetrics()
			return
		case <-ticker.C:
			a.sendMetrics()
		}
	}
}

func (a *Agent) sendMetrics() {
	m := a.collector.GetMetrics()
	metrics := m.ToMetricsSlice()

	if len(metrics) == 0 {
		return
	}

	log.Printf("Sending %d metrics to worker pool", len(metrics))

	timeout := time.After(5 * time.Second)
	done := make(chan bool, 1)

	go func() {
		for _, metric := range metrics {
			if !a.pool.Submit(metric) {
				log.Printf("Failed to submit metric %s (pool stopped or queue full)", metric.ID)
			}
		}
		done <- true
	}()

	select {
	case <-done:
		log.Printf("All %d metrics submitted successfully", len(metrics))
	case <-timeout:
		log.Printf("Timeout submitting metrics, some may be lost")
	}
}

func (a *Agent) gracefulStop() {
	a.mu.Lock()
	if a.stopped {
		a.mu.Unlock()
		return
	}
	a.stopped = true
	a.mu.Unlock()

	log.Println("Starting graceful shutdown...")

	a.collector.Stop()
	log.Println("Collector stopped")

	log.Println("Sending final metrics before shutdown...")
	a.sendMetrics()

	log.Println("Waiting for all workers to finish...")
	a.pool.Stop()
	log.Println("All workers finished")

	a.wg.Wait()
	log.Println("Report loop finished")

	log.Println("Agent stopped gracefully")
}

func (a *Agent) Stop() {
	log.Println("Stopping agent...")

	select {
	case <-a.stopChan:
	default:
		close(a.stopChan)
	}

	done := make(chan struct{})
	go func() {
		a.gracefulStop()
		close(done)
	}()

	select {
	case <-done:
		log.Println("Agent stopped gracefully")
	case <-time.After(10 * time.Second):
		log.Println("Graceful shutdown timeout, forcing stop")
	}
}
