package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/model"
	"github.com/Eressleep/metrics-tpl/pkg/retry"
	"github.com/mailru/easyjson"
)

type Sender struct {
	serverAddr     string
	reportInterval time.Duration
	collector      *Collector
	client         *http.Client
	stopChan       chan struct{}
	useGzip        bool
	retryConfig    *retry.Config
}

func NewSender(serverAddr string, reportInterval time.Duration, collector *Collector) *Sender {
	return &Sender{
		serverAddr:     serverAddr,
		reportInterval: reportInterval,
		collector:      collector,
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:    100,
				IdleConnTimeout: 90 * time.Second,
			},
		},
		stopChan:    make(chan struct{}),
		useGzip:     true,
		retryConfig: retry.DefaultConfig(),
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

	batch := make([]model.Metrics, 0, len(metrics.GetAllGauges())+1)

	for name, value := range metrics.GetAllGauges() {
		val := value
		batch = append(batch, model.Metrics{
			ID:    name,
			MType: model.Gauge,
			Value: &val,
		})
	}

	pollCount := metrics.GetPollCount()
	batch = append(batch, model.Metrics{
		ID:    "PollCount",
		MType: model.Counter,
		Delta: &pollCount,
	})

	s.sendBatchWithRetry(batch)
}

func (s *Sender) sendBatchWithRetry(batch []model.Metrics) {
	ctx := context.Background()

	err := retry.Do(ctx, func() error {
		return s.sendBatch(batch)
	}, s.retryConfig)

	if err != nil {
		fmt.Printf("Failed to send batch after retries: %v\n", err)
	}
}

func (s *Sender) sendBatch(batch []model.Metrics) error {
	url := fmt.Sprintf("http://%s/updates", s.serverAddr)

	data, err := easyjson.Marshal(batch)
	if err != nil {
		return fmt.Errorf("error marshaling batch: %w", err)
	}

	var bodyReader io.Reader
	var contentEncoding string

	if s.useGzip {
		compressedData, err := compressData(data)
		if err != nil {
			return fmt.Errorf("error compressing batch: %w", err)
		}
		bodyReader = bytes.NewReader(compressedData)
		contentEncoding = "gzip"
	} else {
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(http.MethodPost, url, bodyReader)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if contentEncoding != "" {
		req.Header.Set("Content-Encoding", contentEncoding)
	}
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending batch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
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
