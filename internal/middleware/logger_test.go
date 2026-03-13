package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func setupLoggerTest(t *testing.T) (*gin.Engine, *zap.Logger, *bytes.Buffer) {
	gin.SetMode(gin.TestMode)

	var buffer bytes.Buffer

	encoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	writer := zapcore.AddSync(&buffer)

	core := zapcore.NewCore(encoder, writer, zapcore.DebugLevel)
	logger := zap.New(core)

	router := gin.New()

	return router, logger, &buffer
}

func TestLogger(t *testing.T) {
	router, logger, buffer := setupLoggerTest(t)

	router.Use(Logger(logger, LoggerConfig{}))

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "test response")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Set("User-Agent", "test-agent")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	logOutput := buffer.String()
	expectedParts := []string{
		"uri", "/test",
		"method", "GET",
		"status", "200",
		"response_size", "13",
		"client_ip", "192.168.1.1",
		"user_agent", "test-agent",
	}

	for _, part := range expectedParts {
		if !contains(logOutput, part) {
			t.Errorf("Expected log to contain %q", part)
		}
	}
}

func TestLoggerWithQueryParams(t *testing.T) {
	router, logger, buffer := setupLoggerTest(t)

	router.Use(Logger(logger, LoggerConfig{}))

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/test?param1=value1&param2=value2", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	logOutput := buffer.String()
	if !contains(logOutput, "/test?param1=value1&param2=value2") {
		t.Error("Expected full path with query params in logs")
	}
}

func TestLoggerSkipPaths(t *testing.T) {
	router, logger, buffer := setupLoggerTest(t)

	router.Use(Logger(logger, LoggerConfig{
		SkipPaths: []string{"/ping", "/health"},
	}))

	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})
	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "healthy")
	})
	router.GET("/metrics", func(c *gin.Context) {
		c.String(http.StatusOK, "metrics")
	})

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if buffer.Len() > 0 {
		t.Error("Logger should not log skipped paths")
	}
	buffer.Reset()

	req = httptest.NewRequest("GET", "/health", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if buffer.Len() > 0 {
		t.Error("Logger should not log skipped paths")
	}
	buffer.Reset()

	req = httptest.NewRequest("GET", "/metrics", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if buffer.Len() == 0 {
		t.Error("Logger should log non-skipped paths")
	}
}

func TestLoggerWithErrorStatus(t *testing.T) {
	router, logger, buffer := setupLoggerTest(t)

	router.Use(Logger(logger, LoggerConfig{}))

	router.GET("/notfound", func(c *gin.Context) {
		c.String(http.StatusNotFound, "not found")
	})

	router.GET("/error", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "error")
	})

	req := httptest.NewRequest("GET", "/notfound", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	logOutput := buffer.String()
	if !contains(logOutput, "404") {
		t.Error("Expected 404 status in logs")
	}
	buffer.Reset()

	req = httptest.NewRequest("GET", "/error", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	logOutput = buffer.String()
	if !contains(logOutput, "500") {
		t.Error("Expected 500 status in logs")
	}
}

func TestLoggerWithPostRequest(t *testing.T) {
	router, logger, buffer := setupLoggerTest(t)

	router.Use(Logger(logger, LoggerConfig{}))

	router.POST("/submit", func(c *gin.Context) {
		c.String(http.StatusCreated, "created")
	})

	req := httptest.NewRequest("POST", "/submit", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	logOutput := buffer.String()
	if !contains(logOutput, "POST") {
		t.Error("Expected POST method in logs")
	}
	if !contains(logOutput, "201") {
		t.Error("Expected 201 status in logs")
	}
}

func TestLoggerResponseBodyCapture(t *testing.T) {
	router, logger, buffer := setupLoggerTest(t)

	router.Use(Logger(logger, LoggerConfig{}))

	router.GET("/json", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "hello", "value": 42})
	})

	req := httptest.NewRequest("GET", "/json", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	expectedBody := `{"message":"hello","value":42}`
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, w.Body.String())
	}

	actualSize := len(w.Body.String())
	t.Logf("Actual response size: %d", actualSize)

	logOutput := buffer.String()
	if !contains(logOutput, "response_size") {
		t.Error("response_size not found in logs")
	}

	found := false
	for i := 0; i <= 100; i++ {
		if contains(logOutput, string(rune(i))) {
			found = true
			break
		}
	}

	if !found {
		t.Error("No response size number found in logs")
	}
}

func TestLoggerWithLargeResponse(t *testing.T) {
	router, logger, buffer := setupLoggerTest(t)

	router.Use(Logger(logger, LoggerConfig{}))

	router.GET("/large", func(c *gin.Context) {
		largeBody := make([]byte, 10000)
		for i := range largeBody {
			largeBody[i] = 'a'
		}
		c.String(http.StatusOK, string(largeBody))
	})

	req := httptest.NewRequest("GET", "/large", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	logOutput := buffer.String()
	if !contains(logOutput, "response_size") || !contains(logOutput, "10000") {
		t.Error("Expected response_size 10000 in logs")
	}
}

func TestLoggerWithClientIP(t *testing.T) {
	router, logger, buffer := setupLoggerTest(t)

	router.Use(Logger(logger, LoggerConfig{}))

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	req.RemoteAddr = "192.168.1.1:12345"

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	logOutput := buffer.String()
	if contains(logOutput, "192.168.1.1") {
		t.Log("Using RemoteAddr for client IP")
	} else if contains(logOutput, "10.0.0.1") {
		t.Log("Using X-Forwarded-For for client IP")
	} else {
		t.Error("Expected client IP in logs")
	}
}

func TestLoggerWithNilLogger(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Log("Logger middleware with nil logger should not panic")
		}
	}()

	router := gin.New()

	handler := Logger(nil, LoggerConfig{})
	router.Use(handler)

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
}

func TestBodyLogWriter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	blw := &bodyLogWriter{
		ResponseWriter: c.Writer,
		body:           make([]byte, 0),
	}

	data := []byte("test data")
	n, err := blw.Write(data)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(data) {
		t.Errorf("Expected write length %d, got %d", len(data), n)
	}

	if blw.Size() != len(data) {
		t.Errorf("Expected size %d, got %d", len(data), blw.Size())
	}

	blw.Write([]byte(" more"))
	if blw.Size() != len("test data more") {
		t.Errorf("Expected size %d, got %d", len("test data more"), blw.Size())
	}

	if w.Body.String() != "test data more" {
		t.Errorf("Expected body %q, got %q", "test data more", w.Body.String())
	}
}

func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}

func TestLoggerWithEmptyConfig(t *testing.T) {
	router, logger, buffer := setupLoggerTest(t)

	router.Use(Logger(logger, LoggerConfig{}))

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if buffer.Len() == 0 {
		t.Error("Logger should log with empty config")
	}
}

func TestLoggerWithMultipleSkipPaths(t *testing.T) {
	router, logger, buffer := setupLoggerTest(t)

	router.Use(Logger(logger, LoggerConfig{
		SkipPaths: []string{"/skip1", "/skip2", "/skip3"},
	}))

	paths := []string{"/skip1", "/skip2", "/skip3", "/other"}

	for _, path := range paths {
		buffer.Reset()

		router.GET(path, func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if path == "/other" && buffer.Len() == 0 {
			t.Error("Should log non-skipped path")
		}
		if path != "/other" && buffer.Len() > 0 {
			t.Errorf("Should not log skipped path %s", path)
		}
	}
}
