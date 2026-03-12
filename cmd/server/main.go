package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
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

func main() {
	logger, err := zap.NewProduction() // Используем production конфигурацию
	if err != nil {
		log.Fatalf("Не удалось инициализировать логгер: %v", err)
	}
	defer logger.Sync() // Гарантируем сброс буферов при завершении

	serverAddr := flag.String("a", ":8080", "адрес эндпоинта HTTP-сервера")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Использование: %s [флаги]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Флаги:\n")
		fmt.Fprintf(os.Stderr, "  -a=<ЗНАЧЕНИЕ>   адрес эндпоинта HTTP-сервера (по умолчанию :8080)\n")
		fmt.Fprintf(os.Stderr, "\nПеременные окружения:\n")
		fmt.Fprintf(os.Stderr, "  ADDRESS         адрес эндпоинта HTTP-сервера\n")
	}
	flag.Parse()

	if args := flag.Args(); len(args) > 0 {
		logger.Fatal("Неизвестные аргументы", zap.Strings("args", args))
	}

	finalAddr := flags.GetConfigString(serverAddr, "a", "ADDRESS", ":8080")

	memStorage := storage.NewMemStorage()
	metricsHandler := handlers.NewMetricsHandler(memStorage)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(
		middleware.Logger(logger, middleware.LoggerConfig{
			SkipPaths: []string{"/ping"}, // Пропускаем /ping для уменьшения шума
		}),
		gin.Recovery(),
	)

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

	httpServer := &http.Server{
		Addr:    finalAddr,
		Handler: router,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("Сервер запущен", zap.String("address", finalAddr))
		fmt.Println("Доступные эндпоинты:")
		fmt.Println("  POST   /update/:type/:name/:value")
		fmt.Println("  GET    /value/:type/:name")
		fmt.Println("  GET    /")
		fmt.Println("  GET    /ping")

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Ошибка при запуске сервера", zap.Error(err))
		}
	}()

	<-quit
	logger.Info("Получен сигнал завершения, начинаем graceful shutdown...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Fatal("Ошибка при завершении сервера", zap.Error(err))
	}

	logger.Info("Сервер успешно завершил работу")
}
