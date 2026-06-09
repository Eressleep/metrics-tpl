package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/flags"
	"github.com/Eressleep/metrics-tpl/internal/handlers"
	"github.com/Eressleep/metrics-tpl/internal/server"
	"github.com/Eressleep/metrics-tpl/internal/storage"
)

func main() {
	addr := flag.String("a", "localhost:8080", "server address")
	hashKey := flag.String("k", "", "hash key for signing")
	auditFile := flag.String("audit-file", "", "path to audit log file")
	auditURL := flag.String("audit-url", "", "URL to send audit logs")
	storeFile := flag.String("f", "/tmp/metrics-db.json", "file storage path")
	storeInterval := flag.Int("i", 300, "store interval in seconds")
	restore := flag.Bool("r", true, "restore metrics from file on startup")
	databaseDSN := flag.String("d", "", "database DSN")

	flag.Parse()

	config := &server.Config{
		Addr:      flags.GetConfigString(addr, "a", "ADDRESS", "localhost:8080"),
		Mode:      "release",
		AuditFile: flags.GetConfigString(auditFile, "audit-file", "AUDIT_FILE", ""),
		AuditURL:  flags.GetConfigString(auditURL, "audit-url", "AUDIT_URL", ""),
	}

	key := flags.GetConfigString(hashKey, "k", "KEY", "")

	dsn := flags.GetConfigString(databaseDSN, "d", "DATABASE_DSN", "")

	var store storage.Storage
	var err error

	if dsn != "" {
		store, err = storage.NewDBStorage(dsn, nil)
		if err != nil {
			log.Fatalf("Failed to initialize database storage: %v", err)
		}
	} else {
		intervalSeconds := flags.GetConfigInt(storeInterval, "i", "STORE_INTERVAL", 300)
		storeIntervalDuration := time.Duration(intervalSeconds) * time.Second

		filePath := flags.GetConfigString(storeFile, "f", "FILE_STORAGE_PATH", "/tmp/metrics-db.json")
		restoreValue := flags.GetConfigBool(restore, "r", "RESTORE", true)

		store, err = storage.NewFileStorage(
			filePath,
			storeIntervalDuration,
			restoreValue,
			nil,
		)
		if err != nil {
			log.Fatalf("Failed to initialize storage: %v", err)
		}
	}

	metricsHandler := handlers.NewMetricsHandler(store, key)

	srv := server.New(config, metricsHandler)

	fmt.Printf("Starting server on %s\n", config.Addr)
	if key != "" {
		fmt.Printf("Hash key: %s\n", key)
	}
	if config.AuditFile != "" {
		fmt.Printf("Audit log file: %s\n", config.AuditFile)
	}
	if config.AuditURL != "" {
		fmt.Printf("Audit URL: %s\n", config.AuditURL)
	}

	if err := srv.Run(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
