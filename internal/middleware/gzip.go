package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type gzipWriter struct {
	gin.ResponseWriter
	writer *gzip.Writer
	buffer *bytes.Buffer
}

func (g *gzipWriter) Write(data []byte) (int, error) {
	return g.buffer.Write(data)
}

func (g *gzipWriter) WriteString(s string) (int, error) {
	return g.buffer.WriteString(s)
}

func (g *gzipWriter) WriteHeader(code int) {
	g.ResponseWriter.WriteHeader(code)
}

func (g *gzipWriter) Flush() {
	if g.buffer.Len() > 0 {
		data := g.buffer.Bytes()
		compressedData, err := compressData(data)
		if err != nil {
			g.ResponseWriter.Write(data)
			return
		}

		g.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		g.ResponseWriter.Header().Set("Vary", "Accept-Encoding")

		g.ResponseWriter.Write(compressedData)
		g.buffer.Reset()
	}

	if flusher, ok := g.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)

	_, err := gzipWriter.Write(data)
	if err != nil {
		return nil, err
	}

	if err := gzipWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
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
		}

		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		buffer := &bytes.Buffer{}
		gzWriter := &gzipWriter{
			ResponseWriter: c.Writer,
			writer:         gzip.NewWriter(buffer),
			buffer:         buffer,
		}

		c.Writer = gzWriter

		c.Next()

		contentType := c.Writer.Header().Get("Content-Type")
		if strings.Contains(contentType, "application/json") ||
			strings.Contains(contentType, "text/html") {
			gzWriter.Flush()
		} else {
			c.Writer.Write(gzWriter.buffer.Bytes())
		}
	}
}
