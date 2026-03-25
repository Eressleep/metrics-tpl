package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/handlers"
	"github.com/Eressleep/metrics-tpl/internal/storage"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func setupTest(t *testing.T) (*Server, *storage.MemStorage, *zap.Logger) {
	gin.SetMode(gin.TestMode)

	store := storage.NewMemStorage()
	handler := handlers.NewMetricsHandler(store)
	logger := zaptest.NewLogger(t)

	config := &Config{
		Addr: ":0",
		Mode: gin.TestMode,
	}

	server := NewWithLogger(config, handler, logger)
	return server, store, logger
}

func TestNewDefaultConfig(t *testing.T) {
	config := NewDefaultConfig()

	if config.Addr != ":8080" {
		t.Errorf("Expected default addr :8080, got %s", config.Addr)
	}
	if config.Mode != gin.ReleaseMode {
		t.Errorf("Expected default mode %s, got %s", gin.ReleaseMode, config.Mode)
	}
}

func TestNew(t *testing.T) {
	store := storage.NewMemStorage()
	handler := handlers.NewMetricsHandler(store)
	config := NewDefaultConfig()

	server := New(config, handler)

	if server == nil {
		t.Fatal("Server should not be nil")
	}
	if server.config != config {
		t.Error("Config not set correctly")
	}
	if server.handler != handler {
		t.Error("Handler not set correctly")
	}
	if server.router == nil {
		t.Error("Router should not be nil")
	}
	if server.httpSrv == nil {
		t.Error("HTTPServer should not be nil")
	}
}

func TestNewWithLogger(t *testing.T) {
	store := storage.NewMemStorage()
	handler := handlers.NewMetricsHandler(store)
	config := NewDefaultConfig()
	logger := zaptest.NewLogger(t)

	server := NewWithLogger(config, handler, logger)

	if server.logger != logger {
		t.Error("Logger not set correctly")
	}
}

func TestNewWithRouter(t *testing.T) {
	store := storage.NewMemStorage()
	handler := handlers.NewMetricsHandler(store)
	config := NewDefaultConfig()
	logger := zaptest.NewLogger(t)

	router := gin.New()
	router.Use(gin.Recovery())

	server := NewWithRouter(config, handler, router, logger)

	if server.router != router {
		t.Error("Custom router not set correctly")
	}

	server2 := NewWithRouter(config, handler, nil, logger)
	if server2.router == nil {
		t.Error("Router should be created when nil is provided")
	}
}

func TestServerGetRouter(t *testing.T) {
	server, _, _ := setupTest(t)

	router := server.GetRouter()
	if router == nil {
		t.Error("GetRouter should not return nil")
	}
}

func TestServerGetLogger(t *testing.T) {
	server, _, logger := setupTest(t)

	returnedLogger := server.GetLogger()
	if returnedLogger == nil {
		t.Error("GetLogger should not return nil")
	}

	if returnedLogger == logger {
		t.Log("Logger is the same instance")
	}
}

func TestServerRoutes(t *testing.T) {
	server, _, _ := setupTest(t)
	router := server.GetRouter()

	ts := httptest.NewServer(router)
	defer ts.Close()

	updateURL := ts.URL + "/update/gauge/test_metric/123.45"
	resp, err := http.Post(updateURL, "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %d", resp.StatusCode)
	}

	getURL := ts.URL + "/value/gauge/test_metric"
	resp, err = http.Get(getURL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %d", resp.StatusCode)
	}

	resp, err = http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %d", resp.StatusCode)
	}

	resp, err = http.Get(ts.URL + "/ping")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %d", resp.StatusCode)
	}
}

func TestServerNoRoute(t *testing.T) {
	server, _, _ := setupTest(t)
	router := server.GetRouter()

	ts := httptest.NewServer(router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status NotFound, got %d", resp.StatusCode)
	}
}

func TestServerNotFound(t *testing.T) {
	server, _, _ := setupTest(t)

	ts := httptest.NewServer(server.router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status NotFound (404), got %d", resp.StatusCode)
	}
}

func TestServerRunAndStop(t *testing.T) {
	server, _, logger := setupTest(t)

	server.config.Addr = ":0"

	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Run()
	}()

	time.Sleep(100 * time.Millisecond)

	err := server.Stop()
	if err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}

	select {
	case err := <-errChan:
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("Run returned unexpected error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Run did not return after Stop")
	}

	logger.Info("Test complete")
}

func TestServerStopWithoutRun(t *testing.T) {
	server, _, _ := setupTest(t)

	err := server.Stop()
	if err != nil {
		t.Errorf("Stop should not error when server not running: %v", err)
	}
}

func TestServerConfig(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		mode    string
		wantErr bool
	}{
		{
			name:    "valid config",
			addr:    ":8081",
			mode:    gin.TestMode,
			wantErr: false,
		},
		{
			name:    "empty addr",
			addr:    "",
			mode:    gin.TestMode,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := storage.NewMemStorage()
			handler := handlers.NewMetricsHandler(store)
			logger := zaptest.NewLogger(t)

			config := &Config{
				Addr: tt.addr,
				Mode: tt.mode,
			}

			server := NewWithLogger(config, handler, logger)

			errChan := make(chan error, 1)
			go func() {
				errChan <- server.Run()
			}()

			time.Sleep(50 * time.Millisecond)

			server.Stop()

			select {
			case err := <-errChan:
				if tt.wantErr && err == nil {
					t.Error("Expected error but got nil")
				}
				if !tt.wantErr && err != nil && err != http.ErrServerClosed {
					t.Errorf("Unexpected error: %v", err)
				}
			case <-time.After(200 * time.Millisecond):
				if tt.wantErr {
					t.Error("Expected error but server started")
				}
			}
		})
	}
}

func TestServerMultipleStops(t *testing.T) {
	server, _, _ := setupTest(t)

	err1 := server.Stop()
	err2 := server.Stop()
	err3 := server.Stop()

	if err1 != nil {
		t.Errorf("First stop error: %v", err1)
	}
	if err2 != nil {
		t.Errorf("Second stop error: %v", err2)
	}
	if err3 != nil {
		t.Errorf("Third stop error: %v", err3)
	}
}

func TestServerWithCustomRouter(t *testing.T) {
	store := storage.NewMemStorage()
	handler := handlers.NewMetricsHandler(store)
	logger := zaptest.NewLogger(t)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Header("X-Test", "test-header")
		c.Next()
	})

	config := &Config{
		Addr: ":0",
		Mode: gin.TestMode,
	}

	server := NewWithRouter(config, handler, router, logger)

	ts := httptest.NewServer(server.GetRouter())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/ping")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("X-Test") != "test-header" {
		t.Error("Custom middleware not applied")
	}
}

func TestServerShutdownTimeout(t *testing.T) {
	server, _, logger := setupTest(t)

	server.router.GET("/block", func(c *gin.Context) {
		time.Sleep(2 * time.Second)
		c.String(http.StatusOK, "done")
	})

	ts := httptest.NewServer(server.router)
	defer ts.Close()

	done := make(chan bool)
	go func() {
		resp, err := http.Get(ts.URL + "/block")
		if err != nil {
			t.Errorf("Request failed: %v", err)
			done <- false
			return
		}
		defer resp.Body.Close()
		done <- true
	}()

	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := server.httpSrv.Shutdown(ctx)
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("Expected deadline exceeded or nil, got %v", err)
	}

	<-done
	logger.Info("Shutdown test complete")
}

func TestServerConstructors(t *testing.T) {
	store := storage.NewMemStorage()
	handler := handlers.NewMetricsHandler(store)

	t.Run("New", func(t *testing.T) {
		config := NewDefaultConfig()
		server := New(config, handler)
		if server.logger == nil {
			t.Error("Logger should be created")
		}
	})

	t.Run("NewWithLogger", func(t *testing.T) {
		config := NewDefaultConfig()
		logger := zaptest.NewLogger(t)
		server := NewWithLogger(config, handler, logger)
		if server.logger != logger {
			t.Error("Logger not set correctly")
		}
	})

	t.Run("NewWithRouter", func(t *testing.T) {
		config := NewDefaultConfig()
		logger := zaptest.NewLogger(t)
		router := gin.New()
		server := NewWithRouter(config, handler, router, logger)
		if server.router != router {
			t.Error("Router not set correctly")
		}
	})
}
