package agent

import (
	"crypto/rsa"
	"log"
	"strings"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/model"
	"github.com/Eressleep/metrics-tpl/pkg/metrics"
	"github.com/Eressleep/metrics-tpl/pkg/retry"
)

type Sender struct {
	serverAddr     string
	reportInterval time.Duration
	collector      *Collector
	client         *MetricsClient
	stopChan       chan struct{}
	retryConfig    *retry.Config
}

func NewSender(serverAddr string, reportInterval time.Duration, collector *Collector, hashKey string, publicKey *rsa.PublicKey) *Sender {
	return &Sender{
		serverAddr:     serverAddr,
		reportInterval: reportInterval,
		collector:      collector,
		client:         NewMetricsClient(serverAddr, hashKey, publicKey),
		stopChan:       make(chan struct{}),
		retryConfig:    retry.DefaultConfig(),
	}
}

func (s *Sender) Start() {
	go s.reportLoop()
}

func (s *Sender) Stop() {
	close(s.stopChan)
}

func (s *Sender) reportLoop() {
	ticker := time.NewTicker(s.reportInterval)
	defer ticker.Stop()

	s.sendAllMetrics()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.sendAllMetrics()
		}
	}
}

func (s *Sender) sendAllMetrics() {
	m := s.collector.GetMetrics()

	batch := make([]model.Metrics, 0, len(m.GetAllGauges())+1)

	for name, value := range m.GetAllGauges() {
		val := value
		batch = append(batch, model.Metrics{
			ID:    name,
			MType: model.Gauge,
			Value: &val,
		})
	}

	pollCount := m.GetPollCount()
	batch = append(batch, model.Metrics{
		ID:    "PollCount",
		MType: model.Counter,
		Delta: &pollCount,
	})

	log.Printf("Sending batch of %d metrics", len(batch))

	err := s.client.SendBatch(batch)
	if err != nil {
		if strings.Contains(err.Error(), "hash") || strings.Contains(err.Error(), "400") {
			log.Printf("Batch send failed: %v", err)
		} else {
			log.Printf("Batch send failed, trying individual sends: %v", err)
			s.sendIndividualMetrics(m)
		}
	}
}

func (s *Sender) sendIndividualMetrics(m *metrics.Metrics) {
	log.Printf("Sending metrics individually")

	metricsSlice := m.ToMetricsSlice()
	for _, metric := range metricsSlice {
		if metric.MType == model.Counter && metric.ID == "PollCount" {
			continue
		}

		if err := s.client.SendMetric(metric); err != nil {
			log.Printf("Failed to send metric %s: %v", metric.ID, err)
		}
	}
}
