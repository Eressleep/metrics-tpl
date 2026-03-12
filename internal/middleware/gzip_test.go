package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGzipMiddlewareCompressResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(GzipMiddleware())

	router.GET("/test", func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		c.String(http.StatusOK, `{"message":"hello world"}`)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("Expected Content-Encoding: gzip, got %s", w.Header().Get("Content-Encoding"))
	}

	if w.Header().Get("Vary") != "Accept-Encoding" {
		t.Errorf("Expected Vary: Accept-Encoding, got %s", w.Header().Get("Vary"))
	}

	gzipReader, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzipReader.Close()

	decompressed, err := io.ReadAll(gzipReader)
	if err != nil {
		t.Fatalf("Failed to decompress: %v", err)
	}

	expected := `{"message":"hello world"}`
	if string(decompressed) != expected {
		t.Errorf("Expected %q, got %q", expected, string(decompressed))
	}
}

func TestGzipMiddlewareNoCompressForUnsupportedContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(GzipMiddleware())

	router.GET("/test", func(c *gin.Context) {
		c.Header("Content-Type", "text/plain")
		c.String(http.StatusOK, "hello world")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Should not compress text/plain")
	}

	if w.Body.String() != "hello world" {
		t.Errorf("Expected 'hello world', got %q", w.Body.String())
	}
}

func TestGzipMiddlewareDecompressRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(GzipMiddleware())

	router.POST("/test", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			t.Fatal(err)
		}
		c.Header("Content-Type", "application/json")
		c.String(http.StatusOK, string(body))
	})

	originalData := []byte(`{"message":"hello world"}`)
	var compressedBuf bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressedBuf)
	_, err := gzipWriter.Write(originalData)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter.Close()

	req := httptest.NewRequest("POST", "/test", &compressedBuf)
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	if w.Body.String() != string(originalData) {
		t.Errorf("Expected %q, got %q", string(originalData), w.Body.String())
	}
}

func TestGzipMiddlewareDecompressInvalidGzip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(GzipMiddleware())

	router.POST("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "should not reach here")
	})

	invalidData := []byte("not gzip data")
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(invalidData))
	req.Header.Set("Content-Encoding", "gzip")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest, got %d", w.Code)
	}
}
