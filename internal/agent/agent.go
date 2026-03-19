package agent

import (
	"time"
)

type Config struct {
	ServerAddr     string
	PollInterval   time.Duration
	ReportInterval time.Duration
}

func DefaultConfig() *Config {
	return &Config{
		ServerAddr:     "localhost:8080",
		PollInterval:   2 * time.Second,
		ReportInterval: 10 * time.Second,
	}
}

type Agent struct {
	config    *Config
	collector *Collector
	sender    *Sender
}

func New(config *Config) *Agent {
	collector := NewCollector(config.PollInterval)
	sender := NewSender(config.ServerAddr, config.ReportInterval, collector)

	return &Agent{
		config:    config,
		collector: collector,
		sender:    sender,
	}
}

func (a *Agent) Run() {
	a.collector.Start()
	a.sender.Start()

	// Бесконечное ожидание (можно заменить на graceful shutdown)
	select {}
}

func (a *Agent) Stop() {
	a.collector.Stop()
	a.sender.Stop()
}
