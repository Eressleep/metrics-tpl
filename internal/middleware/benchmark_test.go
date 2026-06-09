package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Eressleep/metrics-tpl/internal/model"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func BenchmarkGzipMiddleware(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	logger := zap.NewNop()

	router := gin.New()
	router.Use(GzipMiddleware())
	router.POST("/update/", func(c *gin.Context) {
		c.JSON(http.StatusOK, model.Metrics{ID: "test", MType: "gauge"})
	})

	metric := model.Metrics{ID: "test", MType: "gauge", Value: float64Ptr(123.45)}
	body, _ := json.Marshal(metric)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept-Encoding", "gzip")
		router.ServeHTTP(w, req)
	}
}

func BenchmarkHashMiddleware(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	logger := zap.NewNop()
	key := "test-key"

	router := gin.New()
	router.Use(HashCheckMiddleware(key, logger))
	router.Use(HashResponseMiddleware(key, logger))
	router.POST("/update/", func(c *gin.Context) {
		c.JSON(http.StatusOK, model.Metrics{ID: "test", MType: "gauge"})
	})

	metric := model.Metrics{ID: "test", MType: "gauge", Value: float64Ptr(123.45)}
	body, _ := json.Marshal(metric)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
	}
}
