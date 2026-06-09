package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/audit"
	"github.com/Eressleep/metrics-tpl/internal/handlers"
	"github.com/Eressleep/metrics-tpl/internal/middleware"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Config struct {
	Addr      string
	Mode      string
	AuditFile string
	AuditURL  string
}

func NewDefaultConfig() *Config {
	return &Config{
		Addr:      ":8080",
		Mode:      gin.ReleaseMode,
		AuditFile: "",
		AuditURL:  "",
	}
}

type Server struct {
	config  *Config
	router  *gin.Engine
	handler *handlers.MetricsHandler
	httpSrv *http.Server
	logger  *zap.Logger
	auditor *audit.Auditor
}

func New(config *Config, metricsHandler *handlers.MetricsHandler) *Server {
	logger, _ := zap.NewProduction()
	return NewWithLogger(config, metricsHandler, logger)
}

func NewWithLogger(config *Config, metricsHandler *handlers.MetricsHandler, logger *zap.Logger) *Server {
	gin.SetMode(config.Mode)

	router := gin.New()
	router.HandleMethodNotAllowed = true

	// Отключаем автоматический редирект для trailing slash
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false

	// Базовые middleware
	router.Use(gin.Logger(), gin.Recovery())

	// Добавляем middleware сжатия
	router.Use(middleware.GzipMiddleware())

	// Добавляем middleware хеширования, если ключ задан
	if metricsHandler.GetHashKey() != "" {
		router.Use(middleware.HashCheckMiddleware(metricsHandler.GetHashKey(), logger))
		router.Use(middleware.HashResponseMiddleware(metricsHandler.GetHashKey(), logger))
	}

	// Добавляем middleware аудита
	auditor := audit.New(config.AuditFile, config.AuditURL, logger)
	if auditor.IsEnabled() {
		logger.Info("Audit logging enabled",
			zap.String("file", config.AuditFile),
			zap.String("url", config.AuditURL))
		router.Use(middleware.AuditMiddleware(auditor, logger))
	}

	// Регистрируем обработчики
	router.POST("/update/:type/:name/:value", metricsHandler.Update)
	router.POST("/update/", metricsHandler.UpdateJSON)
	router.POST("/update", metricsHandler.UpdateJSON)
	router.POST("/updates/", metricsHandler.UpdateBatch)
	router.POST("/updates", metricsHandler.UpdateBatch)
	router.POST("/value/", metricsHandler.GetValueJSON)
	router.POST("/value", metricsHandler.GetValueJSON)
	router.GET("/value/:type/:name", metricsHandler.GetValue)
	router.GET("/", metricsHandler.GetAllMetrics)
	router.GET("/ping", metricsHandler.Ping)
	router.GET("/ping-db", metricsHandler.PingDB)

	router.GET("/debug/pprof/", gin.WrapF(pprof.Index))
	router.GET("/debug/pprof/cmdline", gin.WrapF(pprof.Cmdline))
	router.GET("/debug/pprof/profile", gin.WrapF(pprof.Profile))
	router.GET("/debug/pprof/symbol", gin.WrapF(pprof.Symbol))
	router.GET("/debug/pprof/trace", gin.WrapF(pprof.Trace))
	router.GET("/debug/pprof/heap", gin.WrapF(pprof.Handler("heap").ServeHTTP))
	router.GET("/debug/pprof/goroutine", gin.WrapF(pprof.Handler("goroutine").ServeHTTP))
	router.GET("/debug/pprof/threadcreate", gin.WrapF(pprof.Handler("threadcreate").ServeHTTP))
	router.GET("/debug/pprof/block", gin.WrapF(pprof.Handler("block").ServeHTTP))
	
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
		auditor: auditor,
	}
}

func NewWithRouter(config *Config, metricsHandler *handlers.MetricsHandler, router *gin.Engine, logger *zap.Logger) *Server {
	gin.SetMode(config.Mode)

	if router == nil {
		return NewWithLogger(config, metricsHandler, logger)
	}

	router.HandleMethodNotAllowed = true

	// Отключаем автоматический редирект для trailing slash
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false

	// Добавляем middleware хеширования, если ключ задан
	if metricsHandler.GetHashKey() != "" {
		router.Use(middleware.HashCheckMiddleware(metricsHandler.GetHashKey(), logger))
		router.Use(middleware.HashResponseMiddleware(metricsHandler.GetHashKey(), logger))
	}

	// Добавляем middleware аудита
	auditor := audit.New(config.AuditFile, config.AuditURL, logger)
	if auditor.IsEnabled() {
		logger.Info("Audit logging enabled",
			zap.String("file", config.AuditFile),
			zap.String("url", config.AuditURL))
		router.Use(middleware.AuditMiddleware(auditor, logger))
	}

	// Регистрируем обработчики
	router.POST("/update/:type/:name/:value", metricsHandler.Update)
	router.POST("/update/", metricsHandler.UpdateJSON)
	router.POST("/update", metricsHandler.UpdateJSON)
	router.POST("/updates/", metricsHandler.UpdateBatch)
	router.POST("/updates", metricsHandler.UpdateBatch)
	router.POST("/value/", metricsHandler.GetValueJSON)
	router.POST("/value", metricsHandler.GetValueJSON)
	router.GET("/value/:type/:name", metricsHandler.GetValue)
	router.GET("/", metricsHandler.GetAllMetrics)
	router.GET("/ping", metricsHandler.Ping)
	router.GET("/ping-db", metricsHandler.PingDB)

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
		auditor: auditor,
	}
}

func (s *Server) Run() error {
	s.logger.Info("Starting metrics server",
		zap.String("address", s.config.Addr))

	fmt.Println("Available endpoints:")
	fmt.Println("  POST   /update/:type/:name/:value  - Update metric")
	fmt.Println("  POST   /update                      - Update metric (JSON)")
	fmt.Println("  POST   /updates                     - Update batch metrics")
	fmt.Println("  POST   /value                       - Get metric value (JSON)")
	fmt.Println("  GET    /value/:type/:name           - Get metric value")
	fmt.Println("  GET    /                             - View all metrics")
	fmt.Println("  GET    /ping                         - Health check")
	fmt.Println("  GET    /ping-db                      - Database health check")

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

func (s *Server) GetAuditor() *audit.Auditor {
	return s.auditor
}
