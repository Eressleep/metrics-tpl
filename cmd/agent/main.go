package main

import (
	"log"

	"github.com/Eressleep/metrics-tpl/internal/agent"
)

func main() {
	config := agent.DefaultConfig()

	log.Printf("Starting agent with poll interval: %v, report interval: %v",
		config.PollInterval, config.ReportInterval)
	log.Printf("Sending metrics to: %s", config.ServerAddr)

	agt := agent.New(config)
	agt.Run()
}
