package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/flags"
	"github.com/Eressleep/metrics-tpl/internal/handlers"
	"github.com/Eressleep/metrics-tpl/internal/middleware" // Теперь используется
	"github.com/Eressleep/metrics-tpl/internal/server"
	"github.com/Eressleep/metrics-tpl/internal/storage"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// Инициализируем логгер
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Не удалось инициализировать логгер: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Определяем флаги командной строки
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
		logger.Fatal("Неизвестные аргументы", zap.Strings("args", args))
	}

	finalAddr := flags.GetConfigString(serverAddr, "a", "ADDRESS", ":8080")

	memStorage := storage.NewMemStorage()
	metricsHandler := handlers.NewMetricsHandler(memStorage)

	config := server.NewDefaultConfig()
	config.Addr = finalAddr
	config.Mode = gin.DebugMode

	// Создаем роутер вручную и подключаем middleware
	gin.SetMode(config.Mode)
	router := gin.New()

	// Явно подключаем middleware логирования
	router.Use(middleware.Logger(logger, middleware.LoggerConfig{
		SkipPaths: []string{"/ping"},
	}))

	router.Use(gin.Recovery())

	// Регистрируем обработчики
	router.POST("/update/:type/:name/:value", metricsHandler.Update)
	router.GET("/value/:type/:name", metricsHandler.GetValue)
	router.GET("/", metricsHandler.GetAllMetrics)
	router.GET("/ping", metricsHandler.Ping)

	router.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"error": "endpoint not found"})
	})

	router.NoMethod(func(c *gin.Context) {
		c.JSON(405, gin.H{"error": "method not allowed"})
	})

	httpSrv := &http.Server{
		Addr:    config.Addr,
		Handler: router,
	}

	srv := &server.Server{
		Config:  config,
		Router:  router,
		Handler: metricsHandler,
		HTTPSrv: httpSrv,
		Logger:  logger,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Run(); err != nil {
			logger.Info("Сервер остановлен", zap.Error(err))
		}
	}()

	logger.Info("Сервер запущен", zap.String("address", finalAddr))

	<-quit
	logger.Info("Получен сигнал завершения")

	logger.Info("Ожидание завершения текущих запросов...")
	time.Sleep(1 * time.Second)

	if err := srv.Stop(); err != nil {
		logger.Fatal("Ошибка при остановке сервера", zap.Error(err))
	}

	logger.Info("Сервер успешно завершил работу")
}
