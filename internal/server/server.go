package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/handlers"
	"github.com/gin-gonic/gin"
)

type Config struct {
	Addr string
	Mode string // "debug" или "release"
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
	httpSrv *http.Server // Добавьте это поле
}

func New(config *Config, metricsHandler *handlers.MetricsHandler) *Server {
	gin.SetMode(config.Mode)

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.POST("/update/:type/:name/:value", metricsHandler.Update)

	router.GET("/value/:type/:name", metricsHandler.GetValue)

	router.GET("/", metricsHandler.GetAllMetrics)

	router.GET("/ping", metricsHandler.Ping)

	router.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"error": "endpoint not found"})
	})

	router.HandleMethodNotAllowed = true
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
	}
}

func (s *Server) Run() error {
	fmt.Printf("Starting metrics server on %s...\n", s.config.Addr)
	fmt.Println("Available endpoints:")
	fmt.Println("  POST   /update/:type/:name/:value  - Update metric")
	fmt.Println("  GET    /value/:type/:name          - Get metric value")
	fmt.Println("  GET    /                            - View all metrics")
	fmt.Println("  GET    /ping                         - Health check")

	return s.httpSrv.ListenAndServe()
}

func (s *Server) Stop() error {
	fmt.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpSrv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	fmt.Println("Server stopped gracefully")
	return nil
}
