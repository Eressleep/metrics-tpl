package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/flags"
	"github.com/Eressleep/metrics-tpl/internal/handlers"
	"github.com/Eressleep/metrics-tpl/internal/middleware"
	"github.com/Eressleep/metrics-tpl/internal/server"
	"github.com/Eressleep/metrics-tpl/internal/storage"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const defaultFilePath = "/tmp/metrics-db.json"

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Не удалось инициализировать логгер: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	serverAddr := flag.String("a", ":8080", "адрес эндпоинта HTTP-сервера")
	storeInterval := flag.Int("i", 300, "интервал сохранения метрик на диск (в секундах, 0 - синхронная запись)")
	filePath := flag.String("f", defaultFilePath, "путь до файла для сохранения метрик")
	restore := flag.Bool("r", true, "загружать ранее сохранённые значения из файла при старте")
	databaseDSN := flag.String("d", "", "строка подключения к PostgreSQL")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Использование: %s [флаги]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Флаги:\n")
		fmt.Fprintf(os.Stderr, "  -a=<ЗНАЧЕНИЕ>   адрес эндпоинта HTTP-сервера (по умолчанию :8080)\n")
		fmt.Fprintf(os.Stderr, "  -i=<ЗНАЧЕНИЕ>   интервал сохранения метрик на диск в секундах (по умолчанию 300, 0 - синхронная запись)\n")
		fmt.Fprintf(os.Stderr, "  -f=<ЗНАЧЕНИЕ>   путь до файла для сохранения метрик (по умолчанию %s)\n", defaultFilePath)
		fmt.Fprintf(os.Stderr, "  -r=<ЗНАЧЕНИЕ>   загружать ранее сохранённые значения из файла (по умолчанию true)\n")
		fmt.Fprintf(os.Stderr, "  -d=<ЗНАЧЕНИЕ>   строка подключения к PostgreSQL\n")
		fmt.Fprintf(os.Stderr, "\nПеременные окружения:\n")
		fmt.Fprintf(os.Stderr, "  ADDRESS          адрес эндпоинта HTTP-сервера\n")
		fmt.Fprintf(os.Stderr, "  STORE_INTERVAL   интервал сохранения метрик на диск (секунды)\n")
		fmt.Fprintf(os.Stderr, "  FILE_STORAGE_PATH путь до файла для сохранения метрик\n")
		fmt.Fprintf(os.Stderr, "  RESTORE          загружать ранее сохранённые значения (true/false)\n")
		fmt.Fprintf(os.Stderr, "  DATABASE_DSN     строка подключения к PostgreSQL\n")
	}
	flag.Parse()

	if args := flag.Args(); len(args) > 0 {
		logger.Fatal("Неизвестные аргументы", zap.Strings("args", args))
	}

	finalAddr := flags.GetConfigString(serverAddr, "a", "ADDRESS", ":8080")
	finalStoreInterval := flags.GetConfigInt(storeInterval, "i", "STORE_INTERVAL", 300)
	finalFilePath := flags.GetConfigString(filePath, "f", "FILE_STORAGE_PATH", defaultFilePath)
	finalRestore := flags.GetConfigBool(restore, "r", "RESTORE", true)
	finalDatabaseDSN := flags.GetConfigString(databaseDSN, "d", "DATABASE_DSN", "")

	var memStorage storage.Storage
	var storageType string

	if finalDatabaseDSN != "" {
		logger.Info("Попытка подключения к PostgreSQL")

		dbStorage, err := storage.NewDBStorage(finalDatabaseDSN, logger)
		if err != nil {
			logger.Warn("Не удалось подключиться к PostgreSQL, пробуем другие варианты",
				zap.Error(err))
		} else {
			memStorage = dbStorage
			storageType = "PostgreSQL"
			logger.Info("Используется PostgreSQL хранилище")
		}
	}

	if memStorage == nil && (finalFilePath != "" || flags.IsFlagSet("f") || os.Getenv("FILE_STORAGE_PATH") != "") {
		logger.Info("Используется файловое хранилище",
			zap.String("path", finalFilePath),
			zap.Int("store_interval", finalStoreInterval),
			zap.Bool("restore", finalRestore))

		fileStorage, err := storage.NewFileStorage(
			finalFilePath,
			time.Duration(finalStoreInterval)*time.Second,
			finalRestore,
			logger,
		)
		if err != nil {
			logger.Error("Не удалось создать файловое хранилище", zap.Error(err))
		} else {
			memStorage = fileStorage
			storageType = "File"
		}
	}

	if memStorage == nil {
		logger.Info("Используется in-memory хранилище")
		memStorage = storage.NewMemStorage()
		storageType = "In-Memory"
	}

	logger.Info("Хранилище выбрано",
		zap.String("type", storageType),
		zap.String("address", finalAddr))

	metricsHandler := handlers.NewMetricsHandler(memStorage)

	config := server.NewDefaultConfig()
	config.Addr = finalAddr
	config.Mode = gin.ReleaseMode

	gin.SetMode(config.Mode)
	router := gin.New()

	router.Use(
		middleware.GzipMiddleware(),
		middleware.Logger(logger, middleware.LoggerConfig{
			SkipPaths: []string{"/ping"},
		}),
		gin.Recovery(),
	)

	router.POST("/update/:type/:name/:value", metricsHandler.Update)
	router.GET("/value/:type/:name", metricsHandler.GetValue)
	router.POST("/update", metricsHandler.UpdateJSON)
	router.POST("/value", metricsHandler.GetValueJSON)
	router.GET("/", metricsHandler.GetAllMetrics)
	router.GET("/ping", metricsHandler.PingDB)

	router.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"error": "endpoint not found"})
	})
	router.NoMethod(func(c *gin.Context) {
		c.JSON(405, gin.H{"error": "method not allowed"})
	})

	srv := server.NewWithRouter(config, metricsHandler, router, logger)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("Сервер запущен",
			zap.String("address", finalAddr),
			zap.String("storage_type", storageType))

		fmt.Println("\n=== Metrics Server ===")
		fmt.Printf("Хранилище: %s\n", storageType)
		fmt.Printf("Адрес: %s\n\n", finalAddr)
		fmt.Println("Доступные эндпоинты:")
		fmt.Println("  POST   /update/:type/:name/:value  (text/plain)")
		fmt.Println("  GET    /value/:type/:name")
		fmt.Println("  POST   /update                      (application/json)")
		fmt.Println("  POST   /value                       (application/json)")
		fmt.Println("  GET    /")
		fmt.Println("  GET    /ping                        (health check)")

		if storageType == "File" {
			fmt.Printf("\nФайловое хранилище:\n")
			fmt.Printf("  - Путь: %s\n", finalFilePath)
			fmt.Printf("  - Интервал сохранения: %d сек\n", finalStoreInterval)
			fmt.Printf("  - Восстановление при старте: %v\n", finalRestore)
		} else if storageType == "PostgreSQL" {
			fmt.Printf("\nPostgreSQL:\n")
			fmt.Printf("  - DSN: %s\n", finalDatabaseDSN)
		} else {
			fmt.Println("\nIn-Memory хранилище (данные не сохраняются между перезапусками)")
		}
		fmt.Println()

		if err := srv.Run(); err != nil {
			logger.Info("Сервер остановлен", zap.Error(err))
		}
	}()

	<-quit
	logger.Info("Получен сигнал завершения, останавливаем сервер...")

	if closer, ok := memStorage.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			logger.Error("Ошибка при закрытии хранилища", zap.Error(err))
		}
	}

	if err := srv.Stop(); err != nil {
		logger.Fatal("Ошибка при остановке сервера", zap.Error(err))
	}

	logger.Info("Сервер успешно завершил работу")
}
