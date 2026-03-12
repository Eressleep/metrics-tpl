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
	"github.com/Eressleep/metrics-tpl/internal/flags"
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
		fmt.Fprintf(os.Stderr, "\nПеременные окружения:\n")
		fmt.Fprintf(os.Stderr, "  ADDRESS          адрес эндпоинта HTTP-сервера\n")
		fmt.Fprintf(os.Stderr, "  REPORT_INTERVAL  частота отправки метрик (секунды)\n")
		fmt.Fprintf(os.Stderr, "  POLL_INTERVAL    частота опроса метрик (секунды)\n")
	}

	flag.Parse()

	args := flag.Args()
	if len(args) > 0 {
		log.Fatalf("неизвестные аргументы: %v", args)
	}

	// Определяем финальные значения с учетом приоритета используя общие функции
	finalServerAddr := flags.GetConfigString(serverAddr, "a", "ADDRESS", "localhost:8080")
	finalReportInterval := flags.GetConfigInt(reportInterval, "r", "REPORT_INTERVAL", 10)
	finalPollInterval := flags.GetConfigInt(pollInterval, "p", "POLL_INTERVAL", 2)

	config := &agent.Config{
		ServerAddr:     finalServerAddr,
		PollInterval:   time.Duration(finalPollInterval) * time.Second,
		ReportInterval: time.Duration(finalReportInterval) * time.Second,
	}

	log.Printf("Запуск агента с интервалом опроса: %v, интервалом отправки: %v",
		config.PollInterval, config.ReportInterval)
	log.Printf("Отправка метрик на: %s", config.ServerAddr)

	agt := agent.New(config)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Получен сигнал завершения, останавливаем агента...")
		agt.Stop()
		os.Exit(0)
	}()

	agt.Run()
}
