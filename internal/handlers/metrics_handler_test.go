package handlers

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Eressleep/metrics-tpl/internal/model"
	"github.com/Eressleep/metrics-tpl/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/mailru/easyjson"
)

func setupMetricsHandlerTest(t *testing.T) (*gin.Engine, *MetricsHandler, storage.Storage) {
	gin.SetMode(gin.TestMode)
	store := storage.NewMemStorage()
	handler := NewMetricsHandler(store)
	router := gin.New()

	router.POST("/update", handler.UpdateJSON)
	router.POST("/value", handler.GetValueJSON)
	router.POST("/update/:type/:name/:value", handler.Update)
	router.GET("/value/:type/:name", handler.GetValue)
	router.GET("/", handler.GetAllMetrics)
	router.GET("/ping", handler.Ping)

	return router, handler, store
}

func TestUpdateJSON_Success_Gauge(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	value := 123.45
	metric := model.Metrics{
		ID:    "test_gauge",
		MType: model.Gauge,
		Value: &value,
	}

	jsonData, _ := easyjson.Marshal(&metric)
	req := httptest.NewRequest("POST", "/update", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	var response model.Metrics
	if err := easyjson.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}

	if response.ID != "test_gauge" {
		t.Errorf("Expected ID test_gauge, got %s", response.ID)
	}
	if response.MType != model.Gauge {
		t.Errorf("Expected type gauge, got %s", response.MType)
	}
	if response.Value == nil || *response.Value != 123.45 {
		t.Errorf("Expected value 123.45, got %v", response.Value)
	}
}

func TestUpdateJSON_Success_Counter(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	delta := int64(42)
	metric := model.Metrics{
		ID:    "test_counter",
		MType: model.Counter,
		Delta: &delta,
	}

	jsonData, _ := easyjson.Marshal(&metric)
	req := httptest.NewRequest("POST", "/update", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	var response model.Metrics
	if err := easyjson.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}

	if response.ID != "test_counter" {
		t.Errorf("Expected ID test_counter, got %s", response.ID)
	}
	if response.MType != model.Counter {
		t.Errorf("Expected type counter, got %s", response.MType)
	}
	if response.Delta == nil || *response.Delta != 42 {
		t.Errorf("Expected delta 42, got %v", response.Delta)
	}
}

func TestUpdateJSON_InvalidContentType(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	req := httptest.NewRequest("POST", "/update", nil)
	req.Header.Set("Content-Type", "text/plain")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest, got %d", w.Code)
	}
}

func TestUpdateJSON_InvalidJSON(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	invalidJSON := []byte(`{"id": "test", "type": "gauge", "value": }`) // некорректный JSON
	req := httptest.NewRequest("POST", "/update", bytes.NewReader(invalidJSON))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest, got %d", w.Code)
	}
}

func TestUpdateJSON_EmptyID(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	metric := model.Metrics{
		ID:    "",
		MType: model.Gauge,
		Value: func() *float64 { v := 1.0; return &v }(),
	}

	jsonData, _ := easyjson.Marshal(&metric)
	req := httptest.NewRequest("POST", "/update", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest, got %d", w.Code)
	}
}

func TestUpdateJSON_InvalidMetricName(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	metric := model.Metrics{
		ID:    "invalid@name", // спецсимволы не разрешены
		MType: model.Gauge,
		Value: func() *float64 { v := 1.0; return &v }(),
	}

	jsonData, _ := easyjson.Marshal(&metric)
	req := httptest.NewRequest("POST", "/update", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest, got %d", w.Code)
	}
}

func TestUpdateJSON_UnknownType(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	metric := model.Metrics{
		ID:    "test",
		MType: "unknown",
		Value: func() *float64 { v := 1.0; return &v }(),
	}

	jsonData, _ := easyjson.Marshal(&metric)
	req := httptest.NewRequest("POST", "/update", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest, got %d", w.Code)
	}
}

func TestUpdateJSON_CounterWithoutDelta(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	metric := model.Metrics{
		ID:    "test",
		MType: model.Counter,
		// Delta отсутствует
	}

	jsonData, _ := easyjson.Marshal(&metric)
	req := httptest.NewRequest("POST", "/update", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest, got %d", w.Code)
	}
}

func TestUpdateJSON_GaugeWithoutValue(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	metric := model.Metrics{
		ID:    "test",
		MType: model.Gauge,
		// Value отсутствует
	}

	jsonData, _ := easyjson.Marshal(&metric)
	req := httptest.NewRequest("POST", "/update", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest, got %d", w.Code)
	}
}

func TestGetValueJSON_Success_Gauge(t *testing.T) {
	router, _, store := setupMetricsHandlerTest(t)

	store.UpdateGauge("test_gauge", 123.45)

	request := model.Metrics{
		ID:    "test_gauge",
		MType: model.Gauge,
	}

	jsonData, _ := easyjson.Marshal(&request)
	req := httptest.NewRequest("POST", "/value", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	var response model.Metrics
	if err := easyjson.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}

	if response.ID != "test_gauge" {
		t.Errorf("Expected ID test_gauge, got %s", response.ID)
	}
	if response.MType != model.Gauge {
		t.Errorf("Expected type gauge, got %s", response.MType)
	}
	if response.Value == nil || *response.Value != 123.45 {
		t.Errorf("Expected value 123.45, got %v", response.Value)
	}
}

func TestGetValueJSON_Success_Counter(t *testing.T) {
	router, _, store := setupMetricsHandlerTest(t)

	store.UpdateCounter("test_counter", 42)

	request := model.Metrics{
		ID:    "test_counter",
		MType: model.Counter,
	}

	jsonData, _ := easyjson.Marshal(&request)
	req := httptest.NewRequest("POST", "/value", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	var response model.Metrics
	if err := easyjson.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}

	if response.ID != "test_counter" {
		t.Errorf("Expected ID test_counter, got %s", response.ID)
	}
	if response.MType != model.Counter {
		t.Errorf("Expected type counter, got %s", response.MType)
	}
	if response.Delta == nil || *response.Delta != 42 {
		t.Errorf("Expected delta 42, got %v", response.Delta)
	}
}

func TestGetValueJSON_NotFound(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	request := model.Metrics{
		ID:    "nonexistent",
		MType: model.Gauge,
	}

	jsonData, _ := easyjson.Marshal(&request)
	req := httptest.NewRequest("POST", "/value", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status NotFound, got %d", w.Code)
	}
}

func TestGetValueJSON_InvalidContentType(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	req := httptest.NewRequest("POST", "/value", nil)
	req.Header.Set("Content-Type", "text/plain")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest, got %d", w.Code)
	}
}

func TestUpdate_Success_Gauge(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	req := httptest.NewRequest("POST", "/update/gauge/test_gauge/123.45", nil)
	req.Header.Set("Content-Type", "text/plain")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}
}

func TestUpdate_Success_Counter(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	req := httptest.NewRequest("POST", "/update/counter/test_counter/42", nil)
	req.Header.Set("Content-Type", "text/plain")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}
}

func TestUpdate_EmptyMetricName(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	req := httptest.NewRequest("POST", "/update/gauge//123.45", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status NotFound, got %d", w.Code)
	}
}

func TestUpdate_InvalidMetricName(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	req := httptest.NewRequest("POST", "/update/gauge/invalid@name/123.45", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest, got %d", w.Code)
	}
}

func TestUpdate_InvalidContentType(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	req := httptest.NewRequest("POST", "/update/gauge/test/123.45", nil)
	req.Header.Set("Content-Type", "application/xml")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest, got %d", w.Code)
	}
}

func TestUpdate_UnknownMetricType(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	req := httptest.NewRequest("POST", "/update/unknown/test/123.45", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest, got %d", w.Code)
	}
}

func TestUpdate_InvalidCounterValue(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	req := httptest.NewRequest("POST", "/update/counter/test/invalid", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest, got %d", w.Code)
	}
}

func TestUpdate_InvalidGaugeValue(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	req := httptest.NewRequest("POST", "/update/gauge/test/invalid", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest, got %d", w.Code)
	}
}

func TestGetValue_Success_Gauge(t *testing.T) {
	router, _, store := setupMetricsHandlerTest(t)

	store.UpdateGauge("test_gauge", 123.45)

	req := httptest.NewRequest("GET", "/value/gauge/test_gauge", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	body, _ := io.ReadAll(w.Body)
	if string(body) != "123.45" {
		t.Errorf("Expected body '123.45', got '%s'", string(body))
	}
}

func TestGetValue_Success_Counter(t *testing.T) {
	router, _, store := setupMetricsHandlerTest(t)

	store.UpdateCounter("test_counter", 42)

	req := httptest.NewRequest("GET", "/value/counter/test_counter", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	body, _ := io.ReadAll(w.Body)
	if string(body) != "42" {
		t.Errorf("Expected body '42', got '%s'", string(body))
	}
}

func TestGetValue_NotFound_Gauge(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	req := httptest.NewRequest("GET", "/value/gauge/nonexistent", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status NotFound, got %d", w.Code)
	}
}

func TestGetValue_NotFound_Counter(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	req := httptest.NewRequest("GET", "/value/counter/nonexistent", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status NotFound, got %d", w.Code)
	}
}

func TestGetValue_InvalidMetricName(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	req := httptest.NewRequest("GET", "/value/gauge/invalid@name", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest, got %d", w.Code)
	}
}

func TestGetValue_UnknownMetricType(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	req := httptest.NewRequest("GET", "/value/unknown/test", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest, got %d", w.Code)
	}
}

func TestGetAllMetrics_Empty(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	req := httptest.NewRequest("GET", "/", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("Expected Content-Type text/html, got %s", w.Header().Get("Content-Type"))
	}

	body := w.Body.String()
	if !bytes.Contains([]byte(body), []byte("No gauge metrics")) {
		t.Error("Expected 'No gauge metrics' in response")
	}
	if !bytes.Contains([]byte(body), []byte("No counter metrics")) {
		t.Error("Expected 'No counter metrics' in response")
	}
}

func TestPing(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	req := httptest.NewRequest("GET", "/ping", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	body, _ := io.ReadAll(w.Body)
	if string(body) != "pong" {
		t.Errorf("Expected body 'pong', got '%s'", string(body))
	}
}

func TestIsValidMetricName(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"valid_name", true},
		{"valid123", true},
		{"VALID_NAME", true},
		{"name_with_underscores", true},
		{"", false},
		{"a", true},                        // минимальная длина 1
		{string(make([]byte, 101)), false}, // превышение длины
		{"invalid@name", false},
		{"invalid-name", false}, // дефис не разрешен
		{"invalid.name", false}, // точка не разрешена
		{"invalid name", false}, // пробел не разрешен
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidMetricName(tt.name)
			if result != tt.expected {
				t.Errorf("isValidMetricName(%q) = %v, expected %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestHandleCounter(t *testing.T) {
	handler := &MetricsHandler{
		storage: storage.NewMemStorage(),
	}

	tests := []struct {
		name     string
		valueStr string
		wantErr  bool
	}{
		{"valid int", "42", false},
		{"valid negative", "-10", false},
		{"valid zero", "0", false},
		{"invalid string", "abc", true},
		{"invalid float", "123.45", true}, // counter должен быть int
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.handleCounter("test", tt.valueStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleCounter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandleGauge(t *testing.T) {
	handler := &MetricsHandler{
		storage: storage.NewMemStorage(),
	}

	tests := []struct {
		name     string
		valueStr string
		wantErr  bool
	}{
		{"valid float", "123.45", false},
		{"valid int", "42", false},
		{"valid negative", "-10.5", false},
		{"valid zero", "0", false},
		{"invalid string", "abc", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.handleGauge("test", tt.valueStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleGauge() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func BenchmarkUpdateJSON(b *testing.B) {
	router, _, _ := setupMetricsHandlerTest(&testing.T{})

	value := 123.45
	metric := model.Metrics{
		ID:    "bench_metric",
		MType: model.Gauge,
		Value: &value,
	}

	jsonData, _ := easyjson.Marshal(&metric)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/update", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkGetValueJSON(b *testing.B) {
	router, _, store := setupMetricsHandlerTest(&testing.T{})
	store.UpdateGauge("bench_metric", 123.45)

	request := model.Metrics{
		ID:    "bench_metric",
		MType: model.Gauge,
	}

	jsonData, _ := easyjson.Marshal(&request)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/value", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func TestPingDB(t *testing.T) {
	router, _, _ := setupMetricsHandlerTest(t)

	req := httptest.NewRequest("GET", "/ping", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	body, _ := io.ReadAll(w.Body)
	if string(body) != "pong" {
		t.Errorf("Expected body 'pong', got '%s'", string(body))
	}
}
