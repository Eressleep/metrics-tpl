package agent

import (
	"bytes"
	"compress/gzip"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/crypto"
	"github.com/Eressleep/metrics-tpl/internal/model"
	"github.com/Eressleep/metrics-tpl/pkg/hash"
)

type MetricsClient struct {
	serverAddr string
	hashKey    string
	publicKey  *rsa.PublicKey
	httpClient *http.Client
}

func NewMetricsClient(serverAddr, hashKey string, publicKey *rsa.PublicKey) *MetricsClient {
	return &MetricsClient{
		serverAddr: serverAddr,
		hashKey:    hashKey,
		publicKey:  publicKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:    100,
				IdleConnTimeout: 90 * time.Second,
			},
		},
	}
}

func (c *MetricsClient) SendMetric(metric model.Metrics) error {
	url := fmt.Sprintf("http://%s/update", c.serverAddr)
	return c.sendWithRetry(url, metric)
}

func (c *MetricsClient) SendBatch(batch []model.Metrics) error {
	url := fmt.Sprintf("http://%s/updates", c.serverAddr)
	return c.sendWithRetry(url, batch)
}

func (c *MetricsClient) sendWithRetry(url string, data interface{}) error {
	err := c.send(url, data, true)
	if err != nil && c.shouldRetryWithoutHash(err) {
		return c.send(url, data, false)
	}
	return err
}

func (c *MetricsClient) send(url string, data interface{}, withHash bool) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	var finalData []byte
	if c.publicKey != nil {
		encryptedData, err := crypto.EncryptRSA(jsonData, c.publicKey)
		if err != nil {
			return fmt.Errorf("encryption error: %w", err)
		}
		finalData = encryptedData
	} else {
		finalData = jsonData
	}

	compressedData, err := compressData(finalData)
	if err != nil {
		return fmt.Errorf("compress error: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(compressedData))
	if err != nil {
		return fmt.Errorf("create request error: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	if c.publicKey != nil {
		req.Header.Set("X-Encrypted", "true")
	}

	if withHash && c.hashKey != "" {
		req.Header.Set("HashSHA256", hash.ComputeHMAC(compressedData, c.hashKey))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *MetricsClient) shouldRetryWithoutHash(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "hash") || strings.Contains(errStr, "400")
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
