package agent

import (
	"time"

	"github.com/Eressleep/metrics-tpl/pkg/metrics"
)

type Collector struct {
	metrics      *metrics.Metrics
	pollInterval time.Duration
	stopChan     chan struct{}
}

func NewCollector(pollInterval time.Duration) *Collector {
	return &Collector{
		metrics:      metrics.New(),
		pollInterval: pollInterval,
		stopChan:     make(chan struct{}),
	}
}

func (c *Collector) Start() {
	go c.collectLoop()
}

func (c *Collector) Stop() {
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
	c.metrics.UpdateRuntime()

	c.metrics.UpdateRandom()

	c.metrics.IncrementPollCount()
}
