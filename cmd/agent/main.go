package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/agent"
)

func main() {
	serverAddr := flag.String("a", "localhost:8080", "адрес эндпоинта HTTP-сервера")
	reportInterval := flag.Int("r", 10, "частота отправки метрик на сервер (в секундах)")
	pollInterval := flag.Int("p", 2, "частота опроса метрик из пакета runtime (в секундах)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Использование: %s [флаги]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Флаги:\n")
		fmt.Fprintf(os.Stderr, "  -a=<ЗНАЧЕНИЕ>   адрес эндпоинта HTTP-сервера (по умолчанию localhost:8080)\n")
		fmt.Fprintf(os.Stderr, "  -r=<ЗНАЧЕНИЕ>   частота отправки метрик на сервер в секундах (по умолчанию 10)\n")
		fmt.Fprintf(os.Stderr, "  -p=<ЗНАЧЕНИЕ>   частота опроса метрик в секундах (по умолчанию 2)\n")
	}

	flag.Parse()

	args := flag.Args()
	if len(args) > 0 {
		log.Fatalf("неизвестные аргументы: %v", args)
	}

	config := &agent.Config{
		ServerAddr:     *serverAddr,
		PollInterval:   time.Duration(*pollInterval) * time.Second,
		ReportInterval: time.Duration(*reportInterval) * time.Second,
	}

	log.Printf("Запуск агента с интервалом опроса: %v, интервалом отправки: %v",
		config.PollInterval, config.ReportInterval)
	log.Printf("Отправка метрик на: %s", config.ServerAddr)

	agt := agent.New(config)

	go func() {
		<-setupSignalHandler()
		log.Println("Получен сигнал завершения, останавливаем агента...")
		agt.Stop()
		os.Exit(0)
	}()

	agt.Run()
}

func setupSignalHandler() <-chan os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	return c
}
