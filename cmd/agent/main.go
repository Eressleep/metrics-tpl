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
	"github.com/Eressleep/metrics-tpl/internal/build"
	"github.com/Eressleep/metrics-tpl/internal/flags"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	build.Version = buildVersion
	build.Date = buildDate
	build.Commit = buildCommit

	build.PrintBuildInfo()

	serverAddr := flag.String("a", "localhost:8080", "адрес эндпоинта HTTP-сервера")
	reportInterval := flag.Int("r", 10, "частота отправки метрик на сервер (в секундах)")
	pollInterval := flag.Int("p", 2, "частота опроса метрик из пакета runtime (в секундах)")
	hashKey := flag.String("k", "", "ключ для вычисления HMAC-SHA256 хеша")
	rateLimit := flag.Int("l", 1, "количество одновременно исходящих запросов")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Использование: %s [флаги]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Флаги:\n")
		fmt.Fprintf(os.Stderr, "  -a=<ЗНАЧЕНИЕ>   адрес эндпоинта HTTP-сервера (по умолчанию localhost:8080)\n")
		fmt.Fprintf(os.Stderr, "  -r=<ЗНАЧЕНИЕ>   частота отправки метрик на сервер в секундах (по умолчанию 10)\n")
		fmt.Fprintf(os.Stderr, "  -p=<ЗНАЧЕНИЕ>   частота опроса метрик в секундах (по умолчанию 2)\n")
		fmt.Fprintf(os.Stderr, "  -k=<ЗНАЧЕНИЕ>   ключ для вычисления HMAC-SHA256 хеша\n")
		fmt.Fprintf(os.Stderr, "  -l=<ЗНАЧЕНИЕ>   количество одновременно исходящих запросов (по умолчанию 1)\n")
		fmt.Fprintf(os.Stderr, "\nПеременные окружения:\n")
		fmt.Fprintf(os.Stderr, "  ADDRESS          адрес эндпоинта HTTP-сервера\n")
		fmt.Fprintf(os.Stderr, "  REPORT_INTERVAL  частота отправки метрик (секунды)\n")
		fmt.Fprintf(os.Stderr, "  POLL_INTERVAL    частота опроса метрик (секунды)\n")
		fmt.Fprintf(os.Stderr, "  KEY              ключ для вычисления HMAC-SHA256 хеша\n")
		fmt.Fprintf(os.Stderr, "  RATE_LIMIT       количество одновременно исходящих запросов\n")
	}

	flag.Parse()

	args := flag.Args()
	if len(args) > 0 {
		log.Fatalf("неизвестные аргументы: %v", args)
	}

	finalServerAddr := flags.GetConfigString(serverAddr, "a", "ADDRESS", "localhost:8080")
	finalReportInterval := flags.GetConfigInt(reportInterval, "r", "REPORT_INTERVAL", 10)
	finalPollInterval := flags.GetConfigInt(pollInterval, "p", "POLL_INTERVAL", 2)
	finalHashKey := flags.GetConfigString(hashKey, "k", "KEY", "")
	finalRateLimit := flags.GetConfigInt(rateLimit, "l", "RATE_LIMIT", 1)

	config := &agent.Config{
		ServerAddr:     finalServerAddr,
		PollInterval:   time.Duration(finalPollInterval) * time.Second,
		ReportInterval: time.Duration(finalReportInterval) * time.Second,
		HashKey:        finalHashKey,
		RateLimit:      finalRateLimit,
	}

	log.Printf("Запуск агента с интервалом опроса: %v, интервалом отправки: %v",
		config.PollInterval, config.ReportInterval)
	log.Printf("Отправка метрик на: %s", config.ServerAddr)
	log.Printf("Rate limit: %d", config.RateLimit)
	if config.HashKey != "" {
		log.Printf("Используется HMAC-SHA256 подпись с ключом")
	}

	agt := agent.New(config)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		agt.Run()
	}()

	<-sigChan
	log.Println("Получен сигнал завершения, останавливаем агента...")
	agt.Stop()
}
