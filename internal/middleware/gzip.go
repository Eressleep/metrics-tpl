package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type gzipResponseWriter struct {
	gin.ResponseWriter
	gzipWriter *gzip.Writer
}

func (w *gzipResponseWriter) Write(data []byte) (int, error) {
	return w.gzipWriter.Write(data)
}

func (w *gzipResponseWriter) WriteString(s string) (int, error) {
	return w.gzipWriter.Write([]byte(s))
}

func shouldCompress(contentType string) bool {
	return strings.Contains(contentType, "application/json") ||
		strings.Contains(contentType, "text/html")
}

func GzipMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.Contains(c.GetHeader("Content-Encoding"), "gzip") {
			gzipReader, err := gzip.NewReader(c.Request.Body)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "failed to decompress request body"})
				c.Abort()
				return
			}
			defer gzipReader.Close()

			decompressed, err := io.ReadAll(gzipReader)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read decompressed body"})
				c.Abort()
				return
			}

			c.Request.Body = io.NopCloser(bytes.NewReader(decompressed))
			c.Request.ContentLength = int64(len(decompressed))
			c.Request.Header.Set("Content-Encoding", "")
		}

		acceptsGzip := strings.Contains(c.GetHeader("Accept-Encoding"), "gzip")
		if !acceptsGzip {
			c.Next()
			return
		}

		originalWriter := c.Writer

		buffer := &bytes.Buffer{}

		c.Writer = &responseBuffer{
			ResponseWriter: originalWriter,
			buffer:         buffer,
		}

		c.Next()

		contentType := originalWriter.Header().Get("Content-Type")

		if shouldCompress(contentType) && buffer.Len() > 0 {
			originalWriter.Header().Set("Content-Encoding", "gzip")
			originalWriter.Header().Set("Vary", "Accept-Encoding")

			gz := gzip.NewWriter(originalWriter)
			defer gz.Close()

			_, err := gz.Write(buffer.Bytes())
			if err != nil {
				originalWriter.Header().Del("Content-Encoding")
				originalWriter.Write(buffer.Bytes())
			}
		} else {
			if buffer.Len() > 0 {
				originalWriter.Write(buffer.Bytes())
			}
		}
	}
}

type responseBuffer struct {
	gin.ResponseWriter
	buffer *bytes.Buffer
}

func (w *responseBuffer) Write(data []byte) (int, error) {
	return w.buffer.Write(data)
}

func (w *responseBuffer) WriteString(s string) (int, error) {
	return w.buffer.WriteString(s)
}

func (w *responseBuffer) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
}
