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
	"sync"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/model"
	"github.com/Eressleep/metrics-tpl/pkg/hash"
)

type Job struct {
	Metric model.Metrics
}

type WorkerPool struct {
	workers    int
	jobs       chan Job
	serverAddr string
	hashKey    string
	client     *http.Client
	wg         sync.WaitGroup
	stopChan   chan struct{}
	mu         sync.Mutex
	jobsClosed bool
}

func NewWorkerPool(workers int, serverAddr, hashKey string) *WorkerPool {
	if workers < 1 {
		workers = 1
	}
	return &WorkerPool{
		workers:    workers,
		jobs:       make(chan Job, workers*100),
		serverAddr: serverAddr,
		hashKey:    hashKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:    100,
				IdleConnTimeout: 90 * time.Second,
			},
		},
		stopChan:   make(chan struct{}),
		jobsClosed: false,
	}
}

func (p *WorkerPool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	log.Printf("Worker pool started with %d workers", p.workers)
}

func (p *WorkerPool) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	select {
	case <-p.stopChan:
		return
	default:
		close(p.stopChan)
	}

	p.wg.Wait()

	if !p.jobsClosed {
		close(p.jobs)
		p.jobsClosed = true
	}

	log.Println("Worker pool stopped")
}

func (p *WorkerPool) Submit(metric model.Metrics) bool {
	select {
	case p.jobs <- Job{Metric: metric}:
		return true
	case <-p.stopChan:
		return false
	default:
		log.Printf("Job queue full, dropping metric: %s", metric.ID)
		return false
	}
}

func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.stopChan:
			return
		case job, ok := <-p.jobs:
			if !ok {
				return
			}
			p.sendMetric(job.Metric)
		}
	}
}

func (p *WorkerPool) sendMetric(metric model.Metrics) {
	url := fmt.Sprintf("http://%s/update", p.serverAddr)

	data, err := json.Marshal(metric)
	if err != nil {
		log.Printf("Error marshaling metric %s: %v", metric.ID, err)
		return
	}

	compressedData, err := compressData(data)
	if err != nil {
		log.Printf("Error compressing metric %s: %v", metric.ID, err)
		return
	}

	err = p.doRequest(url, compressedData, true)
	if err != nil {
		if strings.Contains(err.Error(), "hash") || strings.Contains(err.Error(), "400") {
			err = p.doRequest(url, compressedData, false)
		}
		if err != nil {
			log.Printf("Failed to send metric %s: %v", metric.ID, err)
		}
	}
}

func (p *WorkerPool) doRequest(url string, body []byte, withHash bool) error {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	if withHash && p.hashKey != "" {
		req.Header.Set("HashSHA256", hash.ComputeHMAC(body, p.hashKey))
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
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
