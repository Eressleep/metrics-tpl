package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Eressleep/metrics-tpl/internal/model"
	"github.com/Eressleep/metrics-tpl/internal/storage"
	"github.com/gin-gonic/gin"
)

func float64Ptr(v float64) *float64 {
	return &v
}

func int64Ptr(v int64) *int64 {
	return &v
}

func setupBenchmarkRouter() (*gin.Engine, *MetricsHandler) {
	gin.SetMode(gin.ReleaseMode)
	store := storage.NewMemStorage()
	handler := NewMetricsHandler(store, "")

	router := gin.New()
	router.POST("/update/", handler.UpdateJSON)
	router.POST("/updates/", handler.UpdateBatch)
	router.POST("/value/", handler.GetValueJSON)

	return router, handler
}

func BenchmarkUpdateJSON(b *testing.B) {
	router, _ := setupBenchmarkRouter()

	metric := model.Metrics{
		ID:    "test_gauge",
		MType: model.Gauge,
		Value: float64Ptr(123.45),
	}
	body, _ := json.Marshal(metric)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
	}
}

func BenchmarkUpdateBatch(b *testing.B) {
	router, _ := setupBenchmarkRouter()

	batch := []model.Metrics{
		{ID: "metric1", MType: model.Gauge, Value: float64Ptr(1.1)},
		{ID: "metric2", MType: model.Counter, Delta: int64Ptr(10)},
		{ID: "metric3", MType: model.Gauge, Value: float64Ptr(2.2)},
		{ID: "metric4", MType: model.Counter, Delta: int64Ptr(20)},
		{ID: "metric5", MType: model.Gauge, Value: float64Ptr(3.3)},
	}
	body, _ := json.Marshal(batch)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
	}
}

func BenchmarkGetValueJSON(b *testing.B) {
	router, handler := setupBenchmarkRouter()

	handler.storage.UpdateGauge("test_gauge", 123.45)

	request := model.Metrics{
		ID:    "test_gauge",
		MType: model.Gauge,
	}
	body, _ := json.Marshal(request)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/value/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
	}
}
