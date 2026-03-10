package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Eressleep/metrics-tpl/internal/handlers"
	"github.com/Eressleep/metrics-tpl/internal/server"
	"github.com/Eressleep/metrics-tpl/internal/storage"
)

func main() {
	memStorage := storage.NewMemStorage()

	metricsHandler := handlers.NewMetricsHandler(memStorage)

	config := server.NewDefaultConfig()

	srv := server.New(config, metricsHandler)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Run(); err != nil {
			log.Fatal("Server failed:", err)
		}
	}()

	<-quit
	log.Println("Received shutdown signal")

	if err := srv.Stop(); err != nil {
		log.Fatal("Server shutdown failed:", err)
	}

	log.Println("Server exited properly")
}
