package agent

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/model"
	"github.com/mailru/easyjson"
)

type Sender struct {
	serverAddr     string
	reportInterval time.Duration
	collector      *Collector
	client         *http.Client
	stopChan       chan struct{}
	maxRetries     int
	retryDelay     time.Duration
}

func NewSender(serverAddr string, reportInterval time.Duration, collector *Collector) *Sender {
	return &Sender{
		serverAddr:     serverAddr,
		reportInterval: reportInterval,
		collector:      collector,
		client:         &http.Client{Timeout: 5 * time.Second},
		stopChan:       make(chan struct{}),
		maxRetries:     3,
		retryDelay:     1 * time.Second,
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
	metrics := s.collector.GetMetrics()

	for name, value := range metrics.GetAllGauges() {
		metric := model.Metrics{
			ID:    name,
			MType: model.Gauge,
			Value: &value,
		}
		go s.sendMetricWithRetry(metric)
	}

	pollCount := metrics.GetPollCount()
	metric := model.Metrics{
		ID:    "PollCount",
		MType: model.Counter,
		Delta: &pollCount,
	}
	go s.sendMetricWithRetry(metric)
}

func (s *Sender) sendMetricWithRetry(metric model.Metrics) {
	var lastErr error
	for i := 0; i < s.maxRetries; i++ {
		if i > 0 {
			delay := s.retryDelay * time.Duration(1<<uint(i-1))
			time.Sleep(delay)
		}

		err := s.sendMetricJSON(metric)
		if err == nil {
			return
		}
		lastErr = err
	}

	if lastErr != nil {
		fmt.Printf("Failed to send metric %s after %d retries: %v\n",
			metric.ID, s.maxRetries, lastErr)
	}
}

func (s *Sender) sendMetricJSON(metric model.Metrics) error {
	url := fmt.Sprintf("http://%s/update", s.serverAddr)

	data, err := easyjson.Marshal(&metric)
	if err != nil {
		return fmt.Errorf("error marshaling metric %s: %w", metric.ID, err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("error creating request for %s: %w", metric.ID, err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending metric %s: %w", metric.ID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code for %s: %d", metric.ID, resp.StatusCode)
	}

	return nil
}
