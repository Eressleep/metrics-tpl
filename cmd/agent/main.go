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
	"github.com/Eressleep/metrics-tpl/internal/config"
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
	cryptoKeyPath := flag.String("crypto-key", "", "путь до файла с публичным ключом для шифрования")
	configFile := flag.String("c", "", "путь к файлу конфигурации")
	flag.StringVar(configFile, "config", "", "путь к файлу конфигурации")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Использование: %s [флаги]\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Флаги:\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nПеременные окружения:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  ADDRESS          адрес эндпоинта HTTP-сервера\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  REPORT_INTERVAL  частота отправки метрик (секунды)\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  POLL_INTERVAL    частота опроса метрик (секунды)\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  KEY              ключ для хеширования\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  RATE_LIMIT       количество одновременно исходящих запросов\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  CRYPTO_KEY       путь до файла с публичным ключом\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  CONFIG           путь к файлу конфигурации\n")
	}

	flag.Parse()

	args := flag.Args()
	if len(args) > 0 {
		log.Fatalf("неизвестные аргументы: %v", args)
	}

	var fileConfig *config.AgentConfig
	if *configFile != "" {
		var err error
		fileConfig, err = config.LoadAgentConfig(*configFile)
		if err != nil {
			log.Printf("WARNING: Failed to load config file: %v", err)
		} else if fileConfig != nil {
			log.Printf("Loaded config from: %s", *configFile)
		}
	}

	finalServerAddr := flags.GetConfigStringWithFile(serverAddr, "a", "ADDRESS", fileConfig.Address, "localhost:8080")
	finalHashKey := flags.GetConfigStringWithFile(hashKey, "k", "KEY", fileConfig.HashKey, "")
	finalCryptoKey := flags.GetConfigStringWithFile(cryptoKeyPath, "crypto-key", "CRYPTO_KEY", fileConfig.CryptoKey, "")

	var fileReportInterval, filePollInterval, fileRateLimit int
	if fileConfig != nil {
		if fileConfig.ReportInterval != "" {
			if dur, err := time.ParseDuration(fileConfig.ReportInterval); err == nil {
				fileReportInterval = int(dur.Seconds())
			}
		}
		if fileConfig.PollInterval != "" {
			if dur, err := time.ParseDuration(fileConfig.PollInterval); err == nil {
				filePollInterval = int(dur.Seconds())
			}
		}
		fileRateLimit = fileConfig.RateLimit
	}

	finalReportInterval := flags.GetConfigIntWithFile(reportInterval, "r", "REPORT_INTERVAL", fileReportInterval, 10)
	finalPollInterval := flags.GetConfigIntWithFile(pollInterval, "p", "POLL_INTERVAL", filePollInterval, 2)
	finalRateLimit := flags.GetConfigIntWithFile(rateLimit, "l", "RATE_LIMIT", fileRateLimit, 1)

	config := &agent.Config{
		ServerAddr:     finalServerAddr,
		PollInterval:   time.Duration(finalPollInterval) * time.Second,
		ReportInterval: time.Duration(finalReportInterval) * time.Second,
		HashKey:        finalHashKey,
		RateLimit:      finalRateLimit,
		CryptoKeyPath:  finalCryptoKey,
	}

	log.Printf("Запуск агента с интервалом опроса: %v, интервалом отправки: %v",
		config.PollInterval, config.ReportInterval)
	log.Printf("Отправка метрик на: %s", config.ServerAddr)
	log.Printf("Rate limit: %d", config.RateLimit)
	if config.HashKey != "" {
		log.Printf("Используется HMAC-SHA256 подпись с ключом")
	}
	if config.CryptoKeyPath != "" {
		log.Printf("Используется RSA шифрование с ключом: %s", config.CryptoKeyPath)
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
