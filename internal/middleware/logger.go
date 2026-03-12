package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type LoggerConfig struct {
	SkipPaths []string
}

func Logger(logger *zap.Logger, config LoggerConfig) gin.HandlerFunc {
	skipPaths := make(map[string]bool, len(config.SkipPaths))
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *gin.Context) {
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		startTime := time.Now()

		blw := &bodyLogWriter{body: make([]byte, 0), ResponseWriter: c.Writer}
		c.Writer = blw

		c.Next()

		latency := time.Since(startTime)
		statusCode := c.Writer.Status()
		contentLength := blw.Size() // Размер ответа в байтах

		path := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			path = path + "?" + raw
		}

		logger.Info("Incoming request",
			zap.String("uri", path),
			zap.String("method", c.Request.Method),
			zap.Duration("duration", latency),
			zap.Int("status", statusCode),
			zap.Int("response_size", contentLength),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		)
	}
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body []byte
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return w.ResponseWriter.Write(b)
}

func (w *bodyLogWriter) Size() int {
	return len(w.body)
}
