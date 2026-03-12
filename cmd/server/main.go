package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/flags"
	"github.com/Eressleep/metrics-tpl/internal/handlers"
	"github.com/Eressleep/metrics-tpl/internal/server"
	"github.com/Eressleep/metrics-tpl/internal/storage"
	"github.com/gin-gonic/gin"
)

func main() {
	serverAddr := flag.String("a", ":8080", "адрес эндпоинта HTTP-сервера")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Использование: %s [флаги]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Флаги:\n")
		fmt.Fprintf(os.Stderr, "  -a=<ЗНАЧЕНИЕ>   адрес эндпоинта HTTP-сервера (по умолчанию :8080)\n")
		fmt.Fprintf(os.Stderr, "\nПеременные окружения:\n")
		fmt.Fprintf(os.Stderr, "  ADDRESS          адрес эндпоинта HTTP-сервера\n")
	}

	flag.Parse()

	args := flag.Args()
	if len(args) > 0 {
		log.Fatalf("неизвестные аргументы: %v", args)
	}

	finalAddr := flags.GetConfigString(serverAddr, "a", "ADDRESS", ":8080")

	memStorage := storage.NewMemStorage()
	metricsHandler := handlers.NewMetricsHandler(memStorage)

	config := server.NewDefaultConfig()
	config.Addr = finalAddr
	config.Mode = gin.DebugMode // debug

	srv := server.New(config, metricsHandler)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Run(); err != nil {
			log.Printf("Сервер остановлен: %v", err)
		}
	}()

	<-quit
	log.Println("Получен сигнал завершения")

	log.Println("Ожидание завершения текущих запросов...")
	time.Sleep(1 * time.Second)

	if err := srv.Stop(); err != nil {
		log.Fatal("Ошибка при остановке сервера:", err)
	}

	log.Println("Сервер успешно завершил работу")
}
