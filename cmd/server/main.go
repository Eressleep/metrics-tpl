package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/build"
	"github.com/Eressleep/metrics-tpl/internal/config"
	"github.com/Eressleep/metrics-tpl/internal/flags"
	"github.com/Eressleep/metrics-tpl/internal/handlers"
	"github.com/Eressleep/metrics-tpl/internal/server"
	"github.com/Eressleep/metrics-tpl/internal/storage"
	"github.com/Eressleep/metrics-tpl/internal/utils"
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

	addr := flag.String("a", "localhost:8080", "server address")
	hashKey := flag.String("k", "", "hash key for signing")
	auditFile := flag.String("audit-file", "", "path to audit log file")
	auditURL := flag.String("audit-url", "", "URL to send audit logs")
	storeFile := flag.String("f", "/tmp/metrics-db.json", "file storage path")
	storeInterval := flag.Int("i", 300, "store interval in seconds")
	restore := flag.Bool("r", true, "restore metrics from file on startup")
	databaseDSN := flag.String("d", "", "database DSN")
	cryptoKeyPath := flag.String("crypto-key", "", "path to private key file for decryption")
	configFile := flag.String("c", "", "path to configuration file")
	flag.StringVar(configFile, "config", "", "path to configuration file")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Использование: %s [флаги]\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Флаги:\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nПеременные окружения:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  ADDRESS          адрес сервера\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  KEY              ключ для хеширования\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  AUDIT_FILE       путь к файлу аудита\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  AUDIT_URL        URL для отправки аудита\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  FILE_STORAGE_PATH путь к файлу хранилища\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  STORE_INTERVAL   интервал сохранения (секунды)\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  RESTORE          восстанавливать метрики при старте\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  DATABASE_DSN     DSN для подключения к БД\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  CRYPTO_KEY       путь к приватному ключу\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  CONFIG           путь к файлу конфигурации\n")
	}

	flag.Parse()

	var fileConfig *config.ServerConfig
	if *configFile != "" {
		var err error
		fileConfig, err = config.LoadServerConfig(*configFile)
		if err != nil {
			log.Printf("WARNING: Failed to load config file: %v", err)
		} else if fileConfig != nil {
			log.Printf("Loaded config from: %s", *configFile)
		}
	}

	var (
		fileAddress       string
		fileHashKey       string
		fileAuditFile     string
		fileAuditURL      string
		fileStoreFile     string
		fileCryptoKey     string
		fileDatabaseDSN   string
		fileStoreInterval int
		fileRestore       *bool
	)

	if fileConfig != nil {
		fileAddress = fileConfig.Address
		fileHashKey = fileConfig.HashKey
		fileAuditFile = fileConfig.AuditFile
		fileAuditURL = fileConfig.AuditURL
		fileStoreFile = fileConfig.StoreFile
		fileCryptoKey = fileConfig.CryptoKey
		fileDatabaseDSN = fileConfig.DatabaseDSN
		fileRestore = fileConfig.Restore

		if fileConfig.StoreInterval != "" {
			if dur, err := time.ParseDuration(fileConfig.StoreInterval); err == nil {
				fileStoreInterval = int(dur.Seconds())
			}
		}
	}

	finalAddr := flags.GetConfigStringWithFile(addr, "a", "ADDRESS", fileAddress, "localhost:8080")
	finalHashKey := flags.GetConfigStringWithFile(hashKey, "k", "KEY", fileHashKey, "")
	finalAuditFile := flags.GetConfigStringWithFile(auditFile, "audit-file", "AUDIT_FILE", fileAuditFile, "")
	finalAuditURL := flags.GetConfigStringWithFile(auditURL, "audit-url", "AUDIT_URL", fileAuditURL, "")
	finalStoreFile := flags.GetConfigStringWithFile(storeFile, "f", "FILE_STORAGE_PATH", fileStoreFile, "/tmp/metrics-db.json")
	finalCryptoKey := flags.GetConfigStringWithFile(cryptoKeyPath, "crypto-key", "CRYPTO_KEY", fileCryptoKey, "")
	finalDatabaseDSN := flags.GetConfigStringWithFile(databaseDSN, "d", "DATABASE_DSN", fileDatabaseDSN, "")
	finalStoreInterval := flags.GetConfigIntWithFile(storeInterval, "i", "STORE_INTERVAL", fileStoreInterval, 300)
	finalRestore := flags.GetConfigBoolWithFile(restore, "r", "RESTORE", fileRestore, true)

	config := &server.Config{
		Addr:          finalAddr,
		Mode:          "release",
		AuditFile:     finalAuditFile,
		AuditURL:      finalAuditURL,
		CryptoKeyPath: finalCryptoKey,
	}

	var store storage.Storage
	var err error
	var usingDB bool
	var dbConnectionError error

	if finalDatabaseDSN != "" {
		safeDSN := utils.MaskDSN(finalDatabaseDSN)
		log.Printf("Attempting to connect to database with DSN: %s", safeDSN)

		store, err = storage.NewDBStorage(finalDatabaseDSN, nil)
		if err != nil {
			dbConnectionError = err
			log.Printf("WARNING: Failed to initialize database storage: %v", err)
			log.Printf("WARNING: Falling back to file/memory storage")

			storeIntervalDuration := time.Duration(finalStoreInterval) * time.Second

			store, err = storage.NewFileStorage(finalStoreFile, storeIntervalDuration, finalRestore, nil)
			if err != nil {
				log.Fatalf("Failed to initialize fallback storage: %v", err)
			}
			usingDB = false
		} else {
			usingDB = true
			log.Printf("Database storage initialized successfully")
		}
	} else {
		storeIntervalDuration := time.Duration(finalStoreInterval) * time.Second

		store, err = storage.NewFileStorage(finalStoreFile, storeIntervalDuration, finalRestore, nil)
		if err != nil {
			log.Fatalf("Failed to initialize storage: %v", err)
		}
		usingDB = false
	}

	metricsHandler := handlers.NewMetricsHandler(store, finalHashKey)
	metricsHandler.SetUsingDB(usingDB)

	if dbConnectionError != nil {
		metricsHandler.SetDBConnectionError(dbConnectionError)
	}

	srv := server.New(config, metricsHandler)

	fmt.Printf("Starting server on %s\n", config.Addr)
	fmt.Printf("Using database: %v\n", usingDB)
	if finalHashKey != "" {
		fmt.Printf("Hash key: %s\n", finalHashKey)
	}
	if config.AuditFile != "" {
		fmt.Printf("Audit log file: %s\n", config.AuditFile)
	}
	if config.AuditURL != "" {
		fmt.Printf("Audit URL: %s\n", config.AuditURL)
	}
	if config.CryptoKeyPath != "" {
		fmt.Printf("RSA encryption enabled with private key: %s\n", config.CryptoKeyPath)
	}
	fmt.Printf("Store interval: %d seconds\n", finalStoreInterval)
	fmt.Printf("Store file: %s\n", finalStoreFile)
	fmt.Printf("Restore: %v\n", finalRestore)

	if err := srv.Run(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
