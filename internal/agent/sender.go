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
}

func NewSender(serverAddr string, reportInterval time.Duration, collector *Collector) *Sender {
	return &Sender{
		serverAddr:     serverAddr,
		reportInterval: reportInterval,
		collector:      collector,
		client:         &http.Client{Timeout: 5 * time.Second},
		stopChan:       make(chan struct{}),
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
		s.sendMetric("gauge", name, value)
	}

	s.sendMetric("counter", "PollCount", metrics.GetPollCount())
}

func (s *Sender) sendMetric(metricType, name string, value interface{}) {
	url := fmt.Sprintf("http://%s/update/%s/%s/%v", s.serverAddr, metricType, name, value)

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		fmt.Printf("Error creating request for %s: %v\n", name, err)
		return
	}

	req.Header.Set("Content-Type", "text/plain")

	resp, err := s.client.Do(req)
	if err != nil {
		fmt.Printf("Error sending metric %s: %v\n", name, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Unexpected status code for %s: %d\n", name, resp.StatusCode)
	}
}
