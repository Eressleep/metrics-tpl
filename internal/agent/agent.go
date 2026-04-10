package agent

import (
	"bytes"
	_ "compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/model"
	"github.com/Eressleep/metrics-tpl/pkg/hash"
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
	client    *http.Client
	stopChan  chan struct{}
}

func New(config *Config) *Agent {
	return &Agent{
		config:    config,
		collector: NewCollector(config.PollInterval),
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:    100,
				IdleConnTimeout: 90 * time.Second,
			},
		},
		stopChan: make(chan struct{}),
	}
}

func (a *Agent) Run() {
	a.collector.Start()

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

	a.sendAllMetrics()

	for {
		select {
		case <-a.stopChan:
			return
		case <-ticker.C:
			a.sendAllMetrics()
		}
	}
}

func (a *Agent) sendAllMetrics() {
	m := a.collector.GetMetrics()
	metrics := m.ToMetricsSlice()

	log.Printf("Sending batch of %d metrics", len(metrics))

	err := a.sendBatch(metrics, true)
	if err != nil {
		if strings.Contains(err.Error(), "hash") || strings.Contains(err.Error(), "400") {
			log.Printf("Hash verification failed, retrying without hash")
			err = a.sendBatch(metrics, false)
		}
		if err != nil {
			log.Printf("Batch send failed: %v", err)
		}
	}
}

func (a *Agent) sendBatch(batch []model.Metrics, withHash bool) error {
	url := fmt.Sprintf("http://%s/updates", a.config.ServerAddr)

	data, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("error marshaling batch: %w", err)
	}

	compressedData, err := compressData(data)
	if err != nil {
		return fmt.Errorf("error compressing batch: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(compressedData))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	if withHash && a.config.HashKey != "" {
		req.Header.Set("HashSHA256", hash.ComputeHMAC(compressedData, a.config.HashKey))
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending batch: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	log.Printf("Successfully sent batch, status: %d", resp.StatusCode)
	return nil
}

func (a *Agent) Stop() {
	a.collector.Stop()
	close(a.stopChan)
}
