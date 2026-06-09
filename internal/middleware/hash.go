package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/Eressleep/metrics-tpl/pkg/hash"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func HashCheckMiddleware(key string, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if key == "" {
			c.Next()
			return
		}

		if c.Request.Method == http.MethodGet {
			c.Next()
			return
		}

		path := c.Request.URL.Path

		if !strings.Contains(path, "/update") && !strings.Contains(path, "/updates") {
			c.Next()
			return
		}

		requestHash := c.GetHeader("HashSHA256")
		if requestHash == "" {
			requestHash = c.GetHeader("Hash")
		}

		if strings.EqualFold(requestHash, "none") {
			logger.Debug("Hash header is 'none', skipping check")
			c.Next()
			return
		}

		if requestHash == "" {
			logger.Debug("Hash header is missing, skipping check")
			c.Next()
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			logger.Error("Failed to read request body", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
			c.Abort()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		expectedHash := hash.ComputeHMAC(body, key)

		if requestHash != expectedHash {
			logger.Warn("Hash verification failed",
				zap.String("path", path),
				zap.String("expected", expectedHash),
				zap.String("received", requestHash))
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid hash"})
			c.Abort()
			return
		}

		logger.Debug("Hash verification passed")
		c.Next()
	}
}

func HashResponseMiddleware(key string, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if key == "" {
			c.Next()
			return
		}

		if c.Request.Method != http.MethodPost {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		if !strings.Contains(path, "/update") && !strings.Contains(path, "/updates") {
			c.Next()
			return
		}

		requestHash := c.GetHeader("HashSHA256")
		if requestHash == "" {
			requestHash = c.GetHeader("Hash")
		}
		if strings.EqualFold(requestHash, "none") {
			c.Next()
			return
		}

		writer := &hashResponseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
			key:            key,
			logger:         logger,
		}
		c.Writer = writer

		c.Next()

		if writer.body.Len() > 0 && c.Writer.Status() == http.StatusOK {
			bodyData := writer.body.Bytes()
			responseHash := hash.ComputeHMAC(bodyData, key)
			c.Header("HashSHA256", responseHash)
			logger.Debug("Added hash to response",
				zap.String("hash", responseHash),
				zap.Int("body_size", len(bodyData)))
		}
	}
}

type hashResponseWriter struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	key    string
	logger *zap.Logger
}

func (w *hashResponseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *hashResponseWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

func (w *hashResponseWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
}

func (w *hashResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w *hashResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
