package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Eressleep/metrics-tpl/internal/model"
	"github.com/Eressleep/metrics-tpl/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/mailru/easyjson"
)

func setupJSONTest() (*gin.Engine, storage.Storage) {
	gin.SetMode(gin.TestMode)
	store := storage.NewMemStorage()
	handler := NewMetricsHandler(store, "")
	router := gin.New()

	router.POST("/update", handler.UpdateJSON)
	router.POST("/value", handler.GetValueJSON)

	return router, store
}

func TestUpdateJSONGauge(t *testing.T) {
	router, _ := setupJSONTest()

	value := 123.45
	metric := model.Metrics{
		ID:    "test_gauge",
		MType: model.Gauge,
		Value: &value,
	}

	jsonData, _ := easyjson.Marshal(&metric)
	req, _ := http.NewRequest("POST", "/update", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", w.Code)
	}

	var response model.Metrics
	err := easyjson.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	if response.ID != "test_gauge" {
		t.Errorf("Expected ID 'test_gauge', got '%s'", response.ID)
	}
	if response.MType != model.Gauge {
		t.Errorf("Expected MType 'gauge', got '%s'", response.MType)
	}
	if response.Value == nil || *response.Value != 123.45 {
		t.Errorf("Expected Value 123.45, got %v", response.Value)
	}
}

func TestUpdateJSONCounter(t *testing.T) {
	router, _ := setupJSONTest()

	delta := int64(42)
	metric := model.Metrics{
		ID:    "test_counter",
		MType: model.Counter,
		Delta: &delta,
	}

	jsonData, _ := easyjson.Marshal(&metric)
	req, _ := http.NewRequest("POST", "/update", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", w.Code)
	}

	var response model.Metrics
	err := easyjson.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	if response.ID != "test_counter" {
		t.Errorf("Expected ID 'test_counter', got '%s'", response.ID)
	}
	if response.MType != model.Counter {
		t.Errorf("Expected MType 'counter', got '%s'", response.MType)
	}
	if response.Delta == nil || *response.Delta != 42 {
		t.Errorf("Expected Delta 42, got %v", response.Delta)
	}
}

func TestUpdateJSONInvalidMetric(t *testing.T) {
	router, _ := setupJSONTest()

	tests := []struct {
		name       string
		metric     model.Metrics
		statusCode int
	}{
		{
			name:       "empty ID",
			metric:     model.Metrics{ID: "", MType: model.Gauge, Value: func() *float64 { v := 1.0; return &v }()},
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "invalid type",
			metric:     model.Metrics{ID: "test", MType: "invalid", Value: func() *float64 { v := 1.0; return &v }()},
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "counter without delta",
			metric:     model.Metrics{ID: "test", MType: model.Counter},
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "gauge without value",
			metric:     model.Metrics{ID: "test", MType: model.Gauge},
			statusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, _ := easyjson.Marshal(&tt.metric)
			req, _ := http.NewRequest("POST", "/update", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.statusCode {
				t.Errorf("Expected status %d, got %d", tt.statusCode, w.Code)
			}
		})
	}
}

func TestGetValueJSON(t *testing.T) {
	router, store := setupJSONTest()

	value := 123.45
	err := store.UpdateGauge("test_gauge", value)
	if err != nil {
		t.Fatal(err)
	}

	request := model.Metrics{
		ID:    "test_gauge",
		MType: model.Gauge,
	}

	jsonData, _ := easyjson.Marshal(&request)
	req, _ := http.NewRequest("POST", "/value", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", w.Code)
	}

	var response model.Metrics
	err = easyjson.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	if response.ID != "test_gauge" {
		t.Errorf("Expected ID 'test_gauge', got '%s'", response.ID)
	}
	if response.MType != model.Gauge {
		t.Errorf("Expected MType 'gauge', got '%s'", response.MType)
	}
	if response.Value == nil || *response.Value != 123.45 {
		t.Errorf("Expected Value 123.45, got %v", response.Value)
	}
}

func TestGetValueJSONNotFound(t *testing.T) {
	router, _ := setupJSONTest()

	request := model.Metrics{
		ID:    "nonexistent",
		MType: model.Gauge,
	}

	jsonData, _ := easyjson.Marshal(&request)
	req, _ := http.NewRequest("POST", "/value", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status NotFound, got %v", w.Code)
	}
}
