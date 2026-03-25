package agent

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
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
	useGzip        bool // флаг для включения gzip сжатия
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
		useGzip:        true, // по умолчанию используем gzip
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

func compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)

	_, err := gzipWriter.Write(data)
	if err != nil {
		return nil, fmt.Errorf("error compressing data: %w", err)
	}

	if err := gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("error closing gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

func (s *Sender) sendMetricJSON(metric model.Metrics) error {
	url := fmt.Sprintf("http://%s/update", s.serverAddr)

	data, err := easyjson.Marshal(&metric)
	if err != nil {
		return fmt.Errorf("error marshaling metric %s: %w", metric.ID, err)
	}

	var bodyReader io.Reader
	var contentEncoding string

	if s.useGzip {
		compressedData, err := compressData(data)
		if err != nil {
			return fmt.Errorf("error compressing metric %s: %w", metric.ID, err)
		}
		bodyReader = bytes.NewReader(compressedData)
		contentEncoding = "gzip"
	} else {
		bodyReader = bytes.NewReader(data)
		contentEncoding = ""
	}

	req, err := http.NewRequest(http.MethodPost, url, bodyReader)
	if err != nil {
		return fmt.Errorf("error creating request for %s: %w", metric.ID, err)
	}

	req.Header.Set("Content-Type", "application/json")
	if contentEncoding != "" {
		req.Header.Set("Content-Encoding", contentEncoding)
	}
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending metric %s: %w", metric.ID, err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return fmt.Errorf("error creating gzip reader for response: %w", err)
		}
		defer gzipReader.Close()
		resp.Body = io.NopCloser(gzipReader)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code for %s: %d", metric.ID, resp.StatusCode)
	}

	return nil
}
