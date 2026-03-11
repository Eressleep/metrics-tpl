package main

import (
	"log"

	"github.com/Eressleep/metrics-tpl/internal/handlers"
	"github.com/Eressleep/metrics-tpl/internal/server"
	"github.com/Eressleep/metrics-tpl/internal/storage"
)

func main() {
	memStorage := storage.NewMemStorage()
	metricsHandler := handlers.NewMetricsHandler(memStorage)

	config := server.NewDefaultConfig()
	srv := server.New(config, metricsHandler)

	if err := srv.Run(); err != nil {
		log.Fatal("Server failed:", err)
	}
}
