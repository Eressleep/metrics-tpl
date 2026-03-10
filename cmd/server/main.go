package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/handlers"
	"github.com/Eressleep/metrics-tpl/internal/server"
	"github.com/Eressleep/metrics-tpl/internal/storage"
	"github.com/gin-gonic/gin"
)

func main() {
	memStorage := storage.NewMemStorage()

	metricsHandler := handlers.NewMetricsHandler(memStorage)

	config := server.NewDefaultConfig()

	config.Mode = gin.DebugMode

	srv := server.New(config, metricsHandler)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Run(); err != nil {
			log.Printf("Server stopped: %v", err)
		}
	}()

	<-quit
	log.Println("Received shutdown signal")

	log.Println("Waiting for ongoing requests to complete...")
	time.Sleep(1 * time.Second)

	if err := srv.Stop(); err != nil {
		log.Fatal("Server shutdown failed:", err)
	}

	log.Println("Server exited properly")
}
