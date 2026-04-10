package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGzipMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("compress json response", func(t *testing.T) {
		router := gin.New()
		router.Use(GzipMiddleware())
		router.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "hello"})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/test", nil)
		req.Header.Set("Accept-Encoding", "gzip")

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

		gz, err := gzip.NewReader(w.Body)
		assert.NoError(t, err)
		defer gz.Close()

		decompressed, err := io.ReadAll(gz)
		assert.NoError(t, err)
		assert.Contains(t, string(decompressed), `"message":"hello"`)
	})

	t.Run("no compress for text/plain", func(t *testing.T) {
		router := gin.New()
		router.Use(GzipMiddleware())
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "hello world")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Accept-Encoding", "gzip")

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Empty(t, w.Header().Get("Content-Encoding"))
		assert.Equal(t, "hello world", w.Body.String())
	})

	t.Run("decompress gzip request", func(t *testing.T) {
		router := gin.New()
		router.Use(GzipMiddleware())
		router.POST("/test", func(c *gin.Context) {
			body, err := io.ReadAll(c.Request.Body)
			assert.NoError(t, err)
			c.String(http.StatusOK, string(body))
		})

		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		gz.Write([]byte("compressed data"))
		gz.Close()

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/test", &buf)
		req.Header.Set("Content-Encoding", "gzip")

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "compressed data", w.Body.String())
	})

	t.Run("no gzip without Accept-Encoding header", func(t *testing.T) {
		router := gin.New()
		router.Use(GzipMiddleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "hello"})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Empty(t, w.Header().Get("Content-Encoding"))
	})
}
