package agent

import (
	"log"
	"sync"

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
	wg         sync.WaitGroup
	stopChan   chan struct{}
	mu         sync.Mutex
	stopped    bool
}

func NewWorkerPool(workers int, serverAddr, hashKey string) *WorkerPool {
	if workers < 1 {
		workers = 1
	}
	return &WorkerPool{
		workers:    workers,
		jobs:       make(chan Job, workers*100),
		client:     NewMetricsClient(serverAddr, hashKey),
		serverAddr: serverAddr,
		hashKey:    hashKey,
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
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stopped {
		return
	}

	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	log.Printf("Worker pool started with %d workers", p.workers)
}

func (p *WorkerPool) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stopped {
		return
	}
	p.stopped = true

	close(p.stopChan)
	p.wg.Wait()
	close(p.jobs)

	log.Println("Worker pool stopped")
}

func (p *WorkerPool) Submit(metric model.Metrics) bool {
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return false
	}
	p.mu.Unlock()

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
			if err := p.client.SendMetric(job.Metric); err != nil {
				log.Printf("Worker %d failed to send metric %s: %v", id, job.Metric.ID, err)
			}
		}
	}
}
