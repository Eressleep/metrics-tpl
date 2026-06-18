package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Eressleep/metrics-tpl/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

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
