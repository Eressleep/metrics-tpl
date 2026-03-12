package agent

import (
	"sync/atomic"
	"time"

	"github.com/Eressleep/metrics-tpl/pkg/metrics"
)

type Collector struct {
	metrics      *metrics.Metrics
	pollInterval time.Duration
	stopChan     chan struct{}
	isRunning    atomic.Bool // Флаг для отслеживания состояния
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
		return // Уже запущен
	}
	c.isRunning.Store(true)
	c.stopChan = make(chan struct{})
	go c.collectLoop()
}

func (c *Collector) Stop() {
	if !c.isRunning.Load() {
		return // Уже остановлен
	}
	c.isRunning.Store(false)
	close(c.stopChan)
}

func (c *Collector) GetMetrics() *metrics.Metrics {
	return c.metrics
}

func (c *Collector) collectLoop() {
	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()

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
	c.metrics.UpdateRuntime()
	c.metrics.UpdateRandom()
	c.metrics.IncrementPollCount()
}
