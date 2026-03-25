package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/handlers"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Config struct {
	Addr string
	Mode string
}

func NewDefaultConfig() *Config {
	return &Config{
		Addr: ":8080",
		Mode: gin.ReleaseMode,
	}
}

type Server struct {
	config  *Config
	router  *gin.Engine
	handler *handlers.MetricsHandler
	httpSrv *http.Server
	logger  *zap.Logger
}

func New(config *Config, metricsHandler *handlers.MetricsHandler) *Server {
	logger, _ := zap.NewProduction()
	return NewWithLogger(config, metricsHandler, logger)
}

func NewWithLogger(config *Config, metricsHandler *handlers.MetricsHandler, logger *zap.Logger) *Server {
	gin.SetMode(config.Mode)

	router := gin.New()
	router.HandleMethodNotAllowed = true // Включаем поддержку 405 Method Not Allowed
	router.Use(gin.Logger(), gin.Recovery())

	router.POST("/update/:type/:name/:value", metricsHandler.Update)
	router.GET("/value/:type/:name", metricsHandler.GetValue)
	router.GET("/", metricsHandler.GetAllMetrics)
	router.GET("/ping", metricsHandler.PingDB)

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

	return &Server{
		config:  config,
		router:  router,
		handler: metricsHandler,
		httpSrv: httpSrv,
		logger:  logger,
	}
}

func NewWithRouter(config *Config, metricsHandler *handlers.MetricsHandler, router *gin.Engine, logger *zap.Logger) *Server {
	gin.SetMode(config.Mode)

	if router == nil {
		return NewWithLogger(config, metricsHandler, logger)
	}

	// Убедимся, что включена поддержка 405 даже в кастомном роутере
	router.HandleMethodNotAllowed = true

	httpSrv := &http.Server{
		Addr:    config.Addr,
		Handler: router,
	}

	return &Server{
		config:  config,
		router:  router,
		handler: metricsHandler,
		httpSrv: httpSrv,
		logger:  logger,
	}
}

func (s *Server) Run() error {
	s.logger.Info("Starting metrics server",
		zap.String("address", s.config.Addr))

	fmt.Println("Available endpoints:")
	fmt.Println("  POST   /update/:type/:name/:value  - Update metric")
	fmt.Println("  GET    /value/:type/:name          - Get metric value")
	fmt.Println("  GET    /                            - View all metrics")
	fmt.Println("  GET    /ping                         - Health check")

	return s.httpSrv.ListenAndServe()
}

func (s *Server) Stop() error {
	s.logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpSrv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	s.logger.Info("Server stopped gracefully")
	return nil
}

func (s *Server) GetRouter() *gin.Engine {
	return s.router
}

func (s *Server) GetLogger() *zap.Logger {
	return s.logger
}
