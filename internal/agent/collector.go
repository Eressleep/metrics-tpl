package agent

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/Eressleep/metrics-tpl/pkg/metrics"
)

type Collector struct {
	metrics      *metrics.Metrics
	pollInterval time.Duration
	stopChan     chan struct{}
	isRunning    atomic.Bool
	mu           sync.RWMutex
	wg           sync.WaitGroup
}

func NewCollector(pollInterval time.Duration) *Collector {
	return &Collector{
		metrics:      metrics.New(),
		pollInterval: pollInterval,
		stopChan:     make(chan struct{}),
	}
}

func (c *Collector) Start() {
	if c.isRunning.Load() {
		return
	}
	c.isRunning.Store(true)
	c.stopChan = make(chan struct{})
	c.wg.Add(1)
	go c.collectLoop()
}

func (c *Collector) Stop() {
	if !c.isRunning.CompareAndSwap(true, false) {
		return
	}
	close(c.stopChan)
	c.wg.Wait()
}

func (c *Collector) GetMetrics() *metrics.Metrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.metrics.Copy()
}

func (c *Collector) collectLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()

	c.collect()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.collect()
		}
	}
}

func (c *Collector) collect() {
	if !c.isRunning.Load() {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics.UpdateRuntime()
	c.metrics.UpdateGopsutil()
	c.metrics.UpdateRandom()
	c.metrics.IncrementPollCount()
}
