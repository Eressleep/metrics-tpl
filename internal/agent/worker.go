package agent

import (
	"crypto/rsa"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/model"
)

type Job struct {
	Metric model.Metrics
}

type WorkerPool struct {
	workers    int
	jobs       chan Job
	client     *MetricsClient
	serverAddr string
	hashKey    string
	publicKey  *rsa.PublicKey
	wg         sync.WaitGroup
	stopChan   chan struct{}
	stopped    atomic.Bool
	mu         sync.Mutex
}

func NewWorkerPool(workers int, serverAddr, hashKey string, publicKey *rsa.PublicKey) *WorkerPool {
	if workers < 1 {
		workers = 1
	}
	return &WorkerPool{
		workers:    workers,
		jobs:       make(chan Job, workers*100),
		client:     NewMetricsClient(serverAddr, hashKey, publicKey),
		serverAddr: serverAddr,
		hashKey:    hashKey,
		publicKey:  publicKey,
		stopChan:   make(chan struct{}),
	}
}

func (p *WorkerPool) GetServerAddr() string {
	return p.serverAddr
}

func (p *WorkerPool) GetHashKey() string {
	return p.hashKey
}

func (p *WorkerPool) GetWorkersCount() int {
	return p.workers
}

func (p *WorkerPool) Start() {
	if p.stopped.Load() {
		return
	}

	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	log.Printf("Worker pool started with %d workers", p.workers)
	if p.publicKey != nil {
		log.Printf("Using RSA encryption for metrics")
	}
}

func (p *WorkerPool) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.stopped.CompareAndSwap(false, true) {
		return
	}

	log.Println("Stopping worker pool...")

	close(p.stopChan)

	close(p.jobs)

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All workers finished gracefully")
	case <-time.After(10 * time.Second):
		log.Println("Workers timeout, forcing stop")
	}

	log.Println("Worker pool stopped")
}

func (p *WorkerPool) Submit(metric model.Metrics) bool {
	if p.stopped.Load() {
		return false
	}

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
			for job := range p.jobs {
				p.processJob(id, job)
			}
			return
		case job, ok := <-p.jobs:
			if !ok {
				return
			}
			p.processJob(id, job)
		}
	}
}

func (p *WorkerPool) processJob(id int, job Job) {
	done := make(chan error, 1)
	go func() {
		done <- p.client.SendMetric(job.Metric)
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Printf("Worker %d failed to send metric %s: %v", id, job.Metric.ID, err)
		}
	case <-time.After(5 * time.Second):
		log.Printf("Worker %d timeout sending metric %s", id, job.Metric.ID)
	}
}
