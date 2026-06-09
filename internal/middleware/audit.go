package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/audit"
	"github.com/Eressleep/metrics-tpl/internal/model"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func AuditMiddleware(auditor *audit.Auditor, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if auditor == nil || !auditor.IsEnabled() {
			c.Next()
			return
		}

		path := c.Request.URL.Path

		if !isUpdateEndpoint(path) {
			c.Next()
			return
		}

		var bodyBytes []byte
		if c.Request.Body != nil && (strings.TrimRight(path, "/") == "/update" || strings.TrimRight(path, "/") == "/updates") {
			var err error
			bodyBytes, err = io.ReadAll(c.Request.Body)
			if err != nil {
				logger.Error("Failed to read request body for audit", zap.Error(err))
				c.Next()
				return
			}
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		c.Next()

		if c.Writer.Status() != http.StatusOK {
			return
		}

		metrics := extractMetricNames(c, bodyBytes, logger)
		if len(metrics) == 0 {
			return
		}

		event := audit.Event{
			TS:        time.Now().Unix(),
			Metrics:   metrics,
			IPAddress: c.ClientIP(),
		}

		go func() {
			if err := auditor.LogEvent(event); err != nil {
				logger.Error("Failed to log audit event", zap.Error(err))
			}
		}()
	}
}

func isUpdateEndpoint(path string) bool {
	cleanPath := strings.TrimRight(path, "/")
	if cleanPath == "/update" || cleanPath == "/updates" {
		return true
	}

	if strings.HasPrefix(path, "/update/") {
		parts := strings.Split(strings.TrimPrefix(path, "/update/"), "/")
		return len(parts) == 3
	}

	return false
}

func extractMetricNames(c *gin.Context, bodyBytes []byte, logger *zap.Logger) []string {
	path := c.Request.URL.Path

	if strings.HasPrefix(path, "/update/") {
		parts := strings.Split(strings.TrimPrefix(path, "/update/"), "/")
		if len(parts) >= 2 {
			return []string{parts[1]}
		}
		return nil
	}

	cleanPath := strings.TrimRight(path, "/")

	if cleanPath == "/update" && len(bodyBytes) > 0 {
		var metric model.Metrics
		if err := json.Unmarshal(bodyBytes, &metric); err == nil {
			if metric.ID != "" {
				return []string{metric.ID}
			}
		}
		return nil
	}

	if cleanPath == "/updates" && len(bodyBytes) > 0 {
		var batch []model.Metrics
		if err := json.Unmarshal(bodyBytes, &batch); err == nil {
			metrics := make([]string, 0, len(batch))
			for _, m := range batch {
				if m.ID != "" {
					metrics = append(metrics, m.ID)
				}
			}
			return metrics
		}
		return nil
	}

	return nil
}
