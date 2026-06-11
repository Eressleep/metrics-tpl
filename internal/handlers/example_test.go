package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/Eressleep/metrics-tpl/internal/handlers"
	"github.com/Eressleep/metrics-tpl/internal/model"
	"github.com/Eressleep/metrics-tpl/internal/storage"
	"github.com/gin-gonic/gin"
)

// ExampleMetricsHandler_Update демонстрирует обновление метрики через URL параметры.
func ExampleMetricsHandler_Update() {
	gin.SetMode(gin.ReleaseMode)
	store := storage.NewMemStorage()
	handler := handlers.NewMetricsHandler(store, "")

	router := gin.Default()
	router.POST("/update/:type/:name/:value", handler.Update)

	// Отправляем запрос на обновление gauge метрики
	req := httptest.NewRequest(http.MethodPost, "/update/gauge/temperature/23.5", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	fmt.Println(w.Code)
	// Output: 200
}

// ExampleMetricsHandler_UpdateJSON демонстрирует обновление метрики через JSON.
func ExampleMetricsHandler_UpdateJSON() {
	gin.SetMode(gin.ReleaseMode)
	store := storage.NewMemStorage()
	handler := handlers.NewMetricsHandler(store, "")

	router := gin.Default()
	router.POST("/update/", handler.UpdateJSON)

	// Создаем метрику
	value := 23.5
	metric := model.Metrics{
		ID:    "temperature",
		MType: model.Gauge,
		Value: &value,
	}
	body, _ := json.Marshal(metric)

	// Отправляем запрос
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	fmt.Println(w.Code)
	// Output: 200
}

// ExampleMetricsHandler_UpdateBatch демонстрирует пакетное обновление метрик.
func ExampleMetricsHandler_UpdateBatch() {
	gin.SetMode(gin.ReleaseMode)
	store := storage.NewMemStorage()
	handler := handlers.NewMetricsHandler(store, "")

	router := gin.Default()
	router.POST("/updates/", handler.UpdateBatch)

	// Создаем batch метрик
	v1, v2 := 23.5, 65.0
	d1 := int64(100)
	batch := []model.Metrics{
		{ID: "temperature", MType: model.Gauge, Value: &v1},
		{ID: "humidity", MType: model.Gauge, Value: &v2},
		{ID: "requests", MType: model.Counter, Delta: &d1},
	}
	body, _ := json.Marshal(batch)

	// Отправляем batch запрос
	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	fmt.Println(w.Code)
	// Output: 200
}

// ExampleMetricsHandler_GetValueJSON демонстрирует получение значения метрики через JSON.
func ExampleMetricsHandler_GetValueJSON() {
	gin.SetMode(gin.ReleaseMode)
	store := storage.NewMemStorage()
	store.UpdateGauge("temperature", 23.5)
	handler := handlers.NewMetricsHandler(store, "")

	router := gin.Default()
	router.POST("/value/", handler.GetValueJSON)

	// Создаем запрос
	request := model.Metrics{
		ID:    "temperature",
		MType: model.Gauge,
	}
	body, _ := json.Marshal(request)

	// Отправляем запрос
	req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Читаем ответ
	resp := w.Result()
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	fmt.Println(w.Code)
	fmt.Println(string(respBody))
	// Output:
	// 200
	// {"id":"temperature","type":"gauge","value":23.5}
}

// ExampleMetricsHandler_GetValue демонстрирует получение значения метрики через URL.
func ExampleMetricsHandler_GetValue() {
	gin.SetMode(gin.ReleaseMode)
	store := storage.NewMemStorage()
	store.UpdateGauge("temperature", 23.5)
	handler := handlers.NewMetricsHandler(store, "")

	router := gin.Default()
	router.GET("/value/:type/:name", handler.GetValue)

	// Отправляем запрос
	req := httptest.NewRequest(http.MethodGet, "/value/gauge/temperature", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	fmt.Println(w.Code)
	fmt.Println(w.Body.String())
	// Output:
	// 200
	// 23.5
}

// ExampleMetricsHandler_Ping демонстрирует проверку доступности сервера.
func ExampleMetricsHandler_Ping() {
	gin.SetMode(gin.ReleaseMode)
	store := storage.NewMemStorage()
	handler := handlers.NewMetricsHandler(store, "")

	router := gin.Default()
	router.GET("/ping", handler.Ping)

	// Отправляем запрос
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	fmt.Println(w.Code)
	fmt.Println(w.Body.String())
	// Output:
	// 200
	// pong
}

// ExampleMetricsHandler_GetAllMetrics демонстрирует получение всех метрик в HTML формате.
func ExampleMetricsHandler_GetAllMetrics() {
	gin.SetMode(gin.ReleaseMode)
	store := storage.NewMemStorage()
	store.UpdateGauge("temperature", 23.5)
	handler := handlers.NewMetricsHandler(store, "")

	router := gin.Default()
	router.GET("/", handler.GetAllMetrics)

	// Отправляем запрос
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	fmt.Println(w.Code)
	fmt.Println(w.Header().Get("Content-Type"))
	// Output:
	// 200
	// text/html; charset=utf-8
}
