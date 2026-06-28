package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Eressleep/metrics-tpl/internal/handlers"
	"github.com/Eressleep/metrics-tpl/internal/model"
	"github.com/Eressleep/metrics-tpl/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewServer(t *testing.T) {
	config := NewDefaultConfig()
	config.Addr = ":8081"
	store := storage.NewMemStorage()
	handler := handlers.NewMetricsHandler(store, "")

	srv := New(config, handler)
	assert.NotNil(t, srv)
	assert.Equal(t, config.Addr, srv.config.Addr)
	assert.NotNil(t, srv.router)
	assert.NotNil(t, srv.httpSrv)
}

func TestNewWithLogger(t *testing.T) {
	config := NewDefaultConfig()
	store := storage.NewMemStorage()
	handler := handlers.NewMetricsHandler(store, "")
	logger, _ := zap.NewDevelopment()

	srv := NewWithLogger(config, handler, logger)
	assert.NotNil(t, srv)
	assert.Equal(t, logger, srv.logger)
}

func TestServerRunAndStop(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := NewDefaultConfig()
	config.Addr = "localhost:0"
	store := storage.NewMemStorage()
	handler := handlers.NewMetricsHandler(store, "")

	srv := New(config, handler)
	assert.NotNil(t, srv)

	errChan := make(chan error, 1)
	go func() {
		if err := srv.Run(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	err := srv.Stop()
	assert.NoError(t, err)
}

func TestServerGetRouter(t *testing.T) {
	config := NewDefaultConfig()
	store := storage.NewMemStorage()
	handler := handlers.NewMetricsHandler(store, "")

	srv := New(config, handler)
	router := srv.GetRouter()
	assert.NotNil(t, router)
	assert.Equal(t, srv.router, router)
}

func TestServerGetLogger(t *testing.T) {
	config := NewDefaultConfig()
	store := storage.NewMemStorage()
	handler := handlers.NewMetricsHandler(store, "")
	logger, _ := zap.NewDevelopment()

	srv := NewWithLogger(config, handler, logger)
	assert.Equal(t, logger, srv.GetLogger())
}

func TestServerGetAuditor(t *testing.T) {
	config := NewDefaultConfig()
	config.AuditFile = "/tmp/test-audit.log"
	store := storage.NewMemStorage()
	handler := handlers.NewMetricsHandler(store, "")
	logger, _ := zap.NewDevelopment()

	srv := NewWithLogger(config, handler, logger)
	auditor := srv.GetAuditor()
	assert.NotNil(t, auditor)
	assert.True(t, auditor.IsEnabled())
}

func TestServerRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := storage.NewMemStorage()
	handler := handlers.NewMetricsHandler(store, "")
	config := NewDefaultConfig()
	srv := New(config, handler)

	router := srv.GetRouter()

	tests := []struct {
		name       string
		method     string
		path       string
		body       interface{}
		statusCode int
	}{
		{
			name:       "Update gauge metric",
			method:     "POST",
			path:       "/update/gauge/test/123.45",
			body:       nil,
			statusCode: http.StatusOK,
		},
		{
			name:       "Update counter metric",
			method:     "POST",
			path:       "/update/counter/test/100",
			body:       nil,
			statusCode: http.StatusOK,
		},
		{
			name:       "Get gauge value",
			method:     "GET",
			path:       "/value/gauge/test",
			body:       nil,
			statusCode: http.StatusOK,
		},
		{
			name:       "Ping endpoint",
			method:     "GET",
			path:       "/ping",
			body:       nil,
			statusCode: http.StatusOK,
		},
		{
			name:       "Not found route",
			method:     "GET",
			path:       "/not-found",
			body:       nil,
			statusCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			var err error

			if tt.body != nil {
				bodyBytes, _ := json.Marshal(tt.body)
				req, err = http.NewRequest(tt.method, tt.path, bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, err = http.NewRequest(tt.method, tt.path, nil)
			}
			assert.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.statusCode, w.Code)
		})
	}
}

func TestServerWithHashKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := storage.NewMemStorage()
	handler := handlers.NewMetricsHandler(store, "test-key")
	config := NewDefaultConfig()
	srv := New(config, handler)

	router := srv.GetRouter()
	assert.NotNil(t, router)

	value := 123.45
	metric := model.Metrics{
		ID:    "test",
		MType: model.Gauge,
		Value: &value,
	}
	body, _ := json.Marshal(metric)

	req, _ := http.NewRequest("POST", "/update/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestServerWithAudit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := NewDefaultConfig()
	config.AuditFile = "/tmp/test-audit.log"
	store := storage.NewMemStorage()
	handler := handlers.NewMetricsHandler(store, "")
	logger, _ := zap.NewDevelopment()

	srv := NewWithLogger(config, handler, logger)
	router := srv.GetRouter()

	assert.NotNil(t, srv.GetAuditor())
	assert.True(t, srv.GetAuditor().IsEnabled())

	value := 123.45
	metric := model.Metrics{
		ID:    "audit-test",
		MType: model.Gauge,
		Value: &value,
	}
	body, _ := json.Marshal(metric)

	req, _ := http.NewRequest("POST", "/update/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestServerWithCrypto(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := NewDefaultConfig()
	config.CryptoKeyPath = "/tmp/test-key.pem"
	store := storage.NewMemStorage()
	handler := handlers.NewMetricsHandler(store, "")
	logger, _ := zap.NewDevelopment()

	srv := NewWithLogger(config, handler, logger)
	assert.NotNil(t, srv)
	assert.Nil(t, srv.privateKey)
}

func TestServerShutdown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := NewDefaultConfig()
	config.Addr = "localhost:0"
	store := storage.NewMemStorage()
	handler := handlers.NewMetricsHandler(store, "")

	srv := New(config, handler)

	go func() {
		_ = srv.Run()
	}()

	err := srv.Stop()
	assert.NoError(t, err)
}

func TestServerDefaultConfig(t *testing.T) {
	config := NewDefaultConfig()
	assert.Equal(t, ":8080", config.Addr)
	assert.Equal(t, gin.ReleaseMode, config.Mode)
	assert.Equal(t, "", config.AuditFile)
	assert.Equal(t, "", config.AuditURL)
	assert.Equal(t, "", config.CryptoKeyPath)
}

func TestServerWithCustomRouter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := NewDefaultConfig()
	store := storage.NewMemStorage()
	handler := handlers.NewMetricsHandler(store, "")
	logger, _ := zap.NewDevelopment()

	customRouter := gin.New()
	customRouter.GET("/custom", func(c *gin.Context) {
		c.String(http.StatusOK, "custom")
	})

	srv := NewWithLogger(config, handler, logger)
	assert.NotNil(t, srv)

	srv.router.GET("/custom", func(c *gin.Context) {
		c.String(http.StatusOK, "custom")
	})

	req, _ := http.NewRequest("GET", "/custom", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "custom", w.Body.String())
}
