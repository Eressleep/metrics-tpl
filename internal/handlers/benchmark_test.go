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

	value := 123.45
	metric := model.Metrics{
		ID:    "test_gauge",
		MType: model.Gauge,
		Value: &value,
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

	v1, v3 := 1.1, 2.2
	d2, d4 := int64(10), int64(20)
	batch := []model.Metrics{
		{ID: "metric1", MType: model.Gauge, Value: &v1},
		{ID: "metric2", MType: model.Counter, Delta: &d2},
		{ID: "metric3", MType: model.Gauge, Value: &v3},
		{ID: "metric4", MType: model.Counter, Delta: &d4},
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
