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

// AuditMiddleware creates a middleware that logs successful metric updates to audit
func AuditMiddleware(auditor *audit.Auditor, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip if auditor is not enabled
		if auditor == nil || !auditor.IsEnabled() {
			c.Next()
			return
		}

		path := c.Request.URL.Path

		// Skip non-update endpoints
		if !isUpdateEndpoint(path) {
			c.Next()
			return
		}

		// Capture the request body for later use (only for JSON endpoints)
		var bodyBytes []byte
		if c.Request.Body != nil && (strings.TrimRight(path, "/") == "/update" || strings.TrimRight(path, "/") == "/updates") {
			var err error
			bodyBytes, err = io.ReadAll(c.Request.Body)
			if err != nil {
				logger.Error("Failed to read request body for audit", zap.Error(err))
				c.Next()
				return
			}
			// Restore the body for the next handler
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// Process the request first
		c.Next()

		// Only audit successful requests
		if c.Writer.Status() != http.StatusOK {
			return
		}

		// Extract metric names from the request
		metrics := extractMetricNames(c, bodyBytes, logger)
		if len(metrics) == 0 {
			return
		}

		// Create audit event
		event := audit.Event{
			TS:        time.Now().Unix(),
			Metrics:   metrics,
			IPAddress: c.ClientIP(),
		}

		// Log event asynchronously to not block the response
		go func() {
			if err := auditor.LogEvent(event); err != nil {
				logger.Error("Failed to log audit event", zap.Error(err))
			}
		}()
	}
}

// isUpdateEndpoint checks if the path is a metric update endpoint
func isUpdateEndpoint(path string) bool {
	// JSON update endpoints
	cleanPath := strings.TrimRight(path, "/")
	if cleanPath == "/update" || cleanPath == "/updates" {
		return true
	}

	// URL parameter update endpoint: /update/:type/:name/:value
	if strings.HasPrefix(path, "/update/") {
		parts := strings.Split(strings.TrimPrefix(path, "/update/"), "/")
		return len(parts) == 3
	}

	return false
}

// extractMetricNames extracts metric names from the request
func extractMetricNames(c *gin.Context, bodyBytes []byte, logger *zap.Logger) []string {
	path := c.Request.URL.Path

	// Handle URL parameter updates: /update/:type/:name/:value
	if strings.HasPrefix(path, "/update/") {
		// Parse path parameters
		parts := strings.Split(strings.TrimPrefix(path, "/update/"), "/")
		if len(parts) >= 2 {
			return []string{parts[1]} // parts[1] is the metric name
		}
		return nil
	}

	cleanPath := strings.TrimRight(path, "/")

	// Handle JSON single metric update: POST /update
	if cleanPath == "/update" && len(bodyBytes) > 0 {
		var metric model.Metrics
		if err := json.Unmarshal(bodyBytes, &metric); err == nil {
			if metric.ID != "" {
				return []string{metric.ID}
			}
		}
		return nil
	}

	// Handle JSON batch update: POST /updates
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
