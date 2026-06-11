package agent

import (
	"log"
	"sync"
	"sync/atomic"

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
	stopped    atomic.Bool
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
	if p.stopped.Load() {
		return
	}

	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	log.Printf("Worker pool started with %d workers", p.workers)
}

func (p *WorkerPool) Stop() {
	if !p.stopped.CompareAndSwap(false, true) {
		return
	}

	close(p.stopChan)

	close(p.jobs)

	p.wg.Wait()

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
	if err := p.client.SendMetric(job.Metric); err != nil {
		log.Printf("Worker %d failed to send metric %s: %v", id, job.Metric.ID, err)
	}
}
