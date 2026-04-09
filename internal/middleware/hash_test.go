package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Eressleep/metrics-tpl/pkg/hash"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestHashCheckMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()
	key := "testkey"

	t.Run("valid hash", func(t *testing.T) {
		router := gin.New()
		router.Use(HashCheckMiddleware(key, logger))
		router.POST("/update", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		body := []byte(`{"id":"test","type":"gauge","value":1.23}`)
		req := httptest.NewRequest("POST", "/update", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("HashSHA256", hash.ComputeHMAC(body, key))

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid hash", func(t *testing.T) {
		router := gin.New()
		router.Use(HashCheckMiddleware(key, logger))
		router.POST("/update", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		body := []byte(`{"id":"test","type":"gauge","value":1.23}`)
		req := httptest.NewRequest("POST", "/update", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("HashSHA256", "invalidhash")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("hash header is none", func(t *testing.T) {
		router := gin.New()
		router.Use(HashCheckMiddleware(key, logger))
		router.POST("/update", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		body := []byte(`{"id":"test","type":"gauge","value":1.23}`)
		req := httptest.NewRequest("POST", "/update", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Hash", "none")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("skip GET requests", func(t *testing.T) {
		router := gin.New()
		router.Use(HashCheckMiddleware(key, logger))
		router.GET("/value", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/value", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHashResponseMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()
	key := "testkey"

	t.Run("add hash to response", func(t *testing.T) {
		router := gin.New()
		router.Use(HashResponseMiddleware(key, logger))
		router.POST("/update", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest("POST", "/update", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NotEmpty(t, w.Header().Get("HashSHA256"))
		assert.NotEmpty(t, w.Header().Get("Hash"))
	})

	t.Run("skip GET requests", func(t *testing.T) {
		router := gin.New()
		router.Use(HashResponseMiddleware(key, logger))
		router.GET("/value", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"value": 123})
		})

		req := httptest.NewRequest("GET", "/value", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Empty(t, w.Header().Get("HashSHA256"))
	})
}
