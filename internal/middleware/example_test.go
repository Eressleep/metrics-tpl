package middleware_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/Eressleep/metrics-tpl/internal/middleware"
	"github.com/Eressleep/metrics-tpl/internal/model"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ExampleGzipMiddleware демонстрирует использование gzip middleware.
func ExampleGzipMiddleware() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(middleware.GzipMiddleware())
	router.POST("/update/", func(c *gin.Context) {
		c.JSON(http.StatusOK, model.Metrics{ID: "test", MType: "gauge"})
	})

	// Отправляем запрос с gzip
	value := 123.45
	metric := model.Metrics{ID: "test", MType: "gauge", Value: &value}
	body, _ := json.Marshal(metric)

	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	fmt.Println(w.Code)
	fmt.Println(w.Header().Get("Content-Encoding"))
	// Output:
	// 200
	// gzip
}

// ExampleHashMiddleware демонстрирует использование hash middleware.
func ExampleHashMiddleware() {
	gin.SetMode(gin.ReleaseMode)
	logger := zap.NewNop()
	key := "secret-key"

	router := gin.New()
	router.Use(middleware.HashCheckMiddleware(key, logger))
	router.Use(middleware.HashResponseMiddleware(key, logger))
	router.POST("/update/", func(c *gin.Context) {
		c.JSON(http.StatusOK, model.Metrics{ID: "test", MType: "gauge"})
	})

	// Отправляем запрос с хешем
	value := 123.45
	metric := model.Metrics{ID: "test", MType: "gauge", Value: &value}
	body, _ := json.Marshal(metric)

	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	fmt.Println(w.Code)
	// Output: 200
}
