package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/agent"
)

func getStringFromEnv(flagValue string, envName string, defaultValue string) string {
	if envValue := os.Getenv(envName); envValue != "" {
		return envValue
	}
	if flagValue != defaultValue {
		return flagValue
	}
	return defaultValue
}

func getIntFromEnv(flagValue int, envName string, defaultValue int) int {
	if envValue := os.Getenv(envName); envValue != "" {
		if val, err := strconv.Atoi(envValue); err == nil {
			return val
		}
	}
	if flagValue != defaultValue {
		return flagValue
	}
	return defaultValue
}

func main() {
	serverAddrFlag := flag.String("a", "localhost:8080", "адрес эндпоинта HTTP-сервера")
	reportIntervalFlag := flag.Int("r", 10, "частота отправки метрик на сервер (в секундах)")
	pollIntervalFlag := flag.Int("p", 2, "частота опроса метрик из пакета runtime (в секундах)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Использование: %s [флаги]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Флаги:\n")
		fmt.Fprintf(os.Stderr, "  -a=<ЗНАЧЕНИЕ>   адрес эндпоинта HTTP-сервера (по умолчанию localhost:8080)\n")
		fmt.Fprintf(os.Stderr, "  -r=<ЗНАЧЕНИЕ>   частота отправки метрик на сервер в секундах (по умолчанию 10)\n")
		fmt.Fprintf(os.Stderr, "  -p=<ЗНАЧЕНИЕ>   частота опроса метрик в секундах (по умолчанию 2)\n")
		fmt.Fprintf(os.Stderr, "\nТакже поддерживаются переменные окружения:\n")
		fmt.Fprintf(os.Stderr, "  ADDRESS         адрес эндпоинта HTTP-сервера\n")
		fmt.Fprintf(os.Stderr, "  REPORT_INTERVAL частота отправки метрик (секунды)\n")
		fmt.Fprintf(os.Stderr, "  POLL_INTERVAL   частота опроса метрик (секунды)\n")
	}

	flag.Parse()

	args := flag.Args()
	if len(args) > 0 {
		log.Fatalf("неизвестные аргументы: %v", args)
	}

	serverAddr := getStringFromEnv(*serverAddrFlag, "ADDRESS", "localhost:8080")
	reportInterval := getIntFromEnv(*reportIntervalFlag, "REPORT_INTERVAL", 10)
	pollInterval := getIntFromEnv(*pollIntervalFlag, "POLL_INTERVAL", 2)

	config := &agent.Config{
		ServerAddr:     serverAddr,
		PollInterval:   time.Duration(pollInterval) * time.Second,
		ReportInterval: time.Duration(reportInterval) * time.Second,
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
