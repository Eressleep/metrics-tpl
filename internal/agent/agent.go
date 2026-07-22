package agent

import (
	"crypto/rsa"
	"log"
	"sync"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/crypto"
	"github.com/Eressleep/metrics-tpl/pkg/metrics"
)

// generate:reset
type Config struct {
	ServerAddr     string
	PollInterval   time.Duration
	ReportInterval time.Duration
	HashKey        string
	RateLimit      int
	CryptoKeyPath  string
	GRPCAddress    string
	UseGRPC        bool
}

func DefaultConfig() *Config {
	return &Config{
		ServerAddr:     "localhost:8080",
		PollInterval:   2 * time.Second,
		ReportInterval: 10 * time.Second,
		HashKey:        "",
		RateLimit:      1,
		CryptoKeyPath:  "",
		GRPCAddress:    "localhost:50051",
		UseGRPC:        false,
	}
}

type Agent struct {
	config     *Config
	collector  *Collector
	pool       *WorkerPool
	grpcClient *GRPCClient
	stopChan   chan struct{}
	wg         sync.WaitGroup
	mu         sync.Mutex
	stopped    bool
}

func New(config *Config) *Agent {
	collector := NewCollector(config.PollInterval)

	// Загружаем публичный ключ, если указан путь
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

	// Создаем gRPC клиент, если он используется
	var grpcClient *GRPCClient
	if config.UseGRPC && config.GRPCAddress != "" {
		var err error
		timeout := config.ReportInterval
		if timeout < 5*time.Second {
			timeout = 5 * time.Second
		}
		grpcClient, err = NewGRPCClient(config.GRPCAddress, timeout)
		if err != nil {
			log.Printf("WARNING: Failed to create gRPC client: %v", err)
			log.Printf("Falling back to HTTP client")
			config.UseGRPC = false
		} else {
			log.Printf("Using gRPC client with address: %s", config.GRPCAddress)
		}
	}

	pool := NewWorkerPool(config.RateLimit, config.ServerAddr, config.HashKey, publicKey)

	return &Agent{
		config:     config,
		collector:  collector,
		pool:       pool,
		grpcClient: grpcClient,
		stopChan:   make(chan struct{}),
	}
}

func (a *Agent) Run() {
	a.collector.Start()

	// Если используется gRPC, не запускаем HTTP пул
	if !a.config.UseGRPC {
		a.pool.Start()
	}

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

	// Отправляем метрики сразу при запуске
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
	protoMetrics := m.ToProtoMetrics()

	if len(protoMetrics) == 0 {
		return
	}

	log.Printf("Sending %d metrics", len(protoMetrics))

	// Отправляем через gRPC если включен
	if a.config.UseGRPC && a.grpcClient != nil {
		if err := a.grpcClient.SendMetrics(protoMetrics); err != nil {
			log.Printf("Failed to send metrics via gRPC: %v", err)
			// Пробуем отправить через HTTP как fallback
			a.sendViaHTTP(m)
		}
		return
	}

	// Отправляем через HTTP
	a.sendViaHTTP(m)
}

func (a *Agent) sendViaHTTP(m *metrics.Metrics) {
	metricsSlice := m.ToMetricsSlice()

	// Отправляем все метрики с таймаутом
	timeout := time.After(5 * time.Second)
	done := make(chan bool, 1)

	go func() {
		for _, metric := range metricsSlice {
			if !a.pool.Submit(metric) {
				log.Printf("Failed to submit metric %s (pool stopped or queue full)", metric.ID)
			}
		}
		done <- true
	}()

	select {
	case <-done:
		log.Printf("All %d metrics submitted successfully", len(metricsSlice))
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

	// 1. Останавливаем сбор метрик
	a.collector.Stop()
	log.Println("Collector stopped")

	// 2. Отправляем финальную партию метрик
	log.Println("Sending final metrics before shutdown...")
	a.sendMetrics()

	// 3. Останавливаем HTTP пул воркеров
	if !a.config.UseGRPC {
		log.Println("Waiting for all HTTP workers to finish...")
		a.pool.Stop()
		log.Println("All HTTP workers finished")
	}

	// 4. Закрываем gRPC клиент
	if a.grpcClient != nil {
		log.Println("Closing gRPC client...")
		if err := a.grpcClient.Close(); err != nil {
			log.Printf("Failed to close gRPC client: %v", err)
		}
	}

	// 5. Ждем завершения reportLoop
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

	// Ждем graceful остановки с таймаутом
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
