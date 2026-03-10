package agent

import (
	"log"
	"os"
	"os/signal"
	"syscall"
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

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	<-sigChan
	log.Println("Получен сигнал завершения, останавливаем агент...")

	a.Stop()
}

func (a *Agent) Stop() {
	a.collector.Stop()
	a.sender.Stop()
}
