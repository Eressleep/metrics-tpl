package agent

import (
	"fmt"
	"net/http"
	"time"
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
		go s.sendWithRetry("gauge", name, value)
	}

	go s.sendWithRetry("counter", "PollCount", metrics.GetPollCount())
}

func (s *Sender) sendWithRetry(metricType, name string, value interface{}) {
	var lastErr error
	for i := 0; i < s.maxRetries; i++ {
		if i > 0 {
			delay := s.retryDelay * time.Duration(1<<uint(i-1))
			time.Sleep(delay)
		}

		err := s.sendMetric(metricType, name, value)
		if err == nil {
			return
		}
		lastErr = err

	}

	if lastErr != nil {
		fmt.Printf("Failed to send metric %s after %d retries: %v\n",
			name, s.maxRetries, lastErr)
	}
}

func (s *Sender) sendMetric(metricType, name string, value interface{}) error {
	url := fmt.Sprintf("http://%s/update/%s/%s/%v", s.serverAddr, metricType, name, value)

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("error creating request for %s: %w", name, err)
	}

	req.Header.Set("Content-Type", "text/plain")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending metric %s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code for %s: %d", name, resp.StatusCode)
	}

	return nil
}
