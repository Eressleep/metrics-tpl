package server

import (
	"fmt"
	"net/http"

	"github.com/Eressleep/metrics-tpl/internal/handlers"
)

type Config struct {
	Addr string
}

func NewDefaultConfig() *Config {
	return &Config{
		Addr: ":8080",
	}
}

type Server struct {
	config  *Config
	handler http.Handler
}

func New(config *Config, metricsHandler *handlers.MetricsHandler) *Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/update/", metricsHandler.Update)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "Metrics server is running")
			return
		}
		http.NotFound(w, r)
	})

	return &Server{
		config:  config,
		handler: mux,
	}
}

func (s *Server) Run() error {
	fmt.Printf("Starting metrics server on %s...\n", s.config.Addr)
	return http.ListenAndServe(s.config.Addr, s.handler)
}
