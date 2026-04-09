package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/model"
	"github.com/Eressleep/metrics-tpl/pkg/hash"
	"github.com/Eressleep/metrics-tpl/pkg/metrics"
	"github.com/Eressleep/metrics-tpl/pkg/retry"
)

type Sender struct {
	serverAddr     string
	reportInterval time.Duration
	collector      *Collector
	client         *http.Client
	stopChan       chan struct{}
	useGzip        bool
	retryConfig    *retry.Config
	hashKey        string
}

func NewSender(serverAddr string, reportInterval time.Duration, collector *Collector, hashKey string) *Sender {
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
		hashKey:     hashKey,
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

	err := s.sendBatch(batch, true)
	if err != nil {
		if strings.Contains(err.Error(), "hash") || strings.Contains(err.Error(), "400") {
			log.Printf("Hash verification failed, retrying without hash")
			err = s.sendBatch(batch, false)
		}

		if err != nil {
			log.Printf("Batch send failed, trying individual sends: %v", err)
			s.sendIndividualMetrics(m)
		}
	}
}

func (s *Sender) sendIndividualMetrics(m *metrics.Metrics) {
	log.Printf("Sending metrics individually")

	for name, value := range m.GetAllGauges() {
		val := value
		metric := model.Metrics{
			ID:    name,
			MType: model.Gauge,
			Value: &val,
		}

		err := s.sendSingleMetric(metric, true)
		if err != nil {
			if strings.Contains(err.Error(), "hash") || strings.Contains(err.Error(), "400") {
				log.Printf("Hash verification failed for %s, retrying without hash", name)
				if err2 := s.sendSingleMetric(metric, false); err2 != nil {
					log.Printf("Failed to send metric %s without hash: %v", name, err2)
				}
			} else {
				log.Printf("Failed to send metric %s: %v", name, err)
			}
		}
	}

	pollCount := m.GetPollCount()
	metric := model.Metrics{
		ID:    "PollCount",
		MType: model.Counter,
		Delta: &pollCount,
	}

	err := s.sendSingleMetric(metric, true)
	if err != nil {
		if strings.Contains(err.Error(), "hash") || strings.Contains(err.Error(), "400") {
			log.Printf("Hash verification failed for PollCount, retrying without hash")
			if err2 := s.sendSingleMetric(metric, false); err2 != nil {
				log.Printf("Failed to send PollCount without hash: %v", err2)
			}
		} else {
			log.Printf("Failed to send PollCount: %v", err)
		}
	}
}

func (s *Sender) sendSingleMetric(metric model.Metrics, withHash bool) error {
	url := fmt.Sprintf("http://%s/update", s.serverAddr)

	data, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("error marshaling metric: %w", err)
	}

	var bodyReader io.Reader
	var bodyData []byte

	if s.useGzip {
		compressedData, err := compressData(data)
		if err != nil {
			return fmt.Errorf("error compressing metric: %w", err)
		}
		bodyReader = bytes.NewReader(compressedData)
		bodyData = compressedData
	} else {
		bodyReader = bytes.NewReader(data)
		bodyData = data
	}

	req, err := http.NewRequest(http.MethodPost, url, bodyReader)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if s.useGzip {
		req.Header.Set("Content-Encoding", "gzip")
	}
	req.Header.Set("Accept-Encoding", "gzip")

	if withHash && s.hashKey != "" {
		req.Header.Set("HashSHA256", hash.ComputeHMAC(bodyData, s.hashKey))
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending metric: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (s *Sender) sendBatch(batch []model.Metrics, withHash bool) error {
	url := fmt.Sprintf("http://%s/updates", s.serverAddr)

	data, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("error marshaling batch: %w", err)
	}

	var bodyReader io.Reader
	var contentEncoding string
	var bodyData []byte

	if s.useGzip {
		compressedData, err := compressData(data)
		if err != nil {
			return fmt.Errorf("error compressing batch: %w", err)
		}
		bodyReader = bytes.NewReader(compressedData)
		contentEncoding = "gzip"
		bodyData = compressedData
	} else {
		bodyReader = bytes.NewReader(data)
		bodyData = data
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

	if withHash && s.hashKey != "" {
		req.Header.Set("HashSHA256", hash.ComputeHMAC(bodyData, s.hashKey))
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending batch: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Printf("Server returned error %d: %s", resp.StatusCode, string(respBody))
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	log.Printf("Successfully sent batch, status: %d", resp.StatusCode)
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
