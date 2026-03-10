package server

import (
	"embed"
	"fmt"
	"html/template"

	"github.com/Eressleep/metrics-tpl/internal/handlers"
	"github.com/gin-gonic/gin"
)

var templatesFS embed.FS

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
}

func New(config *Config, metricsHandler *handlers.MetricsHandler) *Server {
	gin.SetMode(config.Mode)

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	tmpl := template.Must(template.New("").
		Funcs(handlers.TemplateFunctions()).
		ParseFS(templatesFS, "templates/*.tmpl"))
	router.SetHTMLTemplate(tmpl)

	api := router.Group("/")
	{
		api.POST("/update/:type/:name/:value", metricsHandler.Update)

		api.GET("/value/:type/:name", metricsHandler.GetValue)

		api.GET("/", metricsHandler.GetAllMetrics)
	}

	router.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"error": "endpoint not found"})
	})

	return &Server{
		config:  config,
		router:  router,
		handler: metricsHandler,
	}
}

func (s *Server) Run() error {
	fmt.Printf("🚀 Starting metrics server on %s...\n", s.config.Addr)
	fmt.Printf("📝 Mode: %s\n", s.config.Mode)
	fmt.Println("\n📌 Available endpoints:")
	fmt.Println("  POST   /update/:type/:name/:value  - Update metric")
	fmt.Println("  GET    /value/:type/:name          - Get metric value")
	fmt.Println("  GET    /                            - View all metrics")
	fmt.Println("\n💡 Examples:")
	fmt.Println("  curl -X POST http://localhost:8080/update/gauge/CPU/45.7")
	fmt.Println("  curl -X POST http://localhost:8080/update/counter/Requests/100")
	fmt.Println("  curl http://localhost:8080/value/gauge/CPU")
	fmt.Println("  curl http://localhost:8080/")

	if err := s.router.Run(s.config.Addr); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

func (s *Server) Stop() error {
	fmt.Println("\n🛑 Shutting down server...")
	// Здесь можно добавить логику graceful shutdown
	return nil
}
