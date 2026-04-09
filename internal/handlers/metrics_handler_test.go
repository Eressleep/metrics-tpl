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
	"github.com/stretchr/testify/assert"
)

func TestUpdateJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("update gauge", func(t *testing.T) {
		memStorage := storage.NewMemStorage()
		handler := NewMetricsHandler(memStorage, "")

		router := gin.New()
		router.POST("/update", handler.UpdateJSON)

		value := 123.45
		metric := model.Metrics{
			ID:    "test_gauge",
			MType: "gauge",
			Value: &value,
		}
		body, _ := json.Marshal(metric)

		req := httptest.NewRequest("POST", "/update", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.Metrics
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, value, *response.Value)
	})

	t.Run("invalid metric name", func(t *testing.T) {
		memStorage := storage.NewMemStorage()
		handler := NewMetricsHandler(memStorage, "")

		router := gin.New()
		router.POST("/update", handler.UpdateJSON)

		value := 123.45
		metric := model.Metrics{
			ID:    "invalid-name!",
			MType: "gauge",
			Value: &value,
		}
		body, _ := json.Marshal(metric)

		req := httptest.NewRequest("POST", "/update", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestGetValue(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("get counter value", func(t *testing.T) {
		memStorage := storage.NewMemStorage()
		memStorage.UpdateCounter("test_counter", 42)
		handler := NewMetricsHandler(memStorage, "")

		router := gin.New()
		router.GET("/value/:type/:name", handler.GetValue)

		req := httptest.NewRequest("GET", "/value/counter/test_counter", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "42", w.Body.String())
	})

	t.Run("metric not found", func(t *testing.T) {
		memStorage := storage.NewMemStorage()
		handler := NewMetricsHandler(memStorage, "")

		router := gin.New()
		router.GET("/value/:type/:name", handler.GetValue)

		req := httptest.NewRequest("GET", "/value/counter/nonexistent", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
