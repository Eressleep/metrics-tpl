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
	skip := make(map[string]bool, len(config.SkipPaths))
	for _, path := range config.SkipPaths {
		skip[path] = true
	}

	return func(c *gin.Context) {
		if skip[c.Request.URL.Path] {
			c.Next()
			return
		}

		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		if raw != "" {
			path = path + "?" + raw
		}

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		size := c.Writer.Size()
		method := c.Request.Method

		logger.Info("Request processed",
			zap.String("uri", path),
			zap.String("method", method),
			zap.Duration("duration", latency),
			zap.Int("status", status),
			zap.Int("size", size),
		)
	}
}
