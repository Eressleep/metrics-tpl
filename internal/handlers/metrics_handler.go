package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/Eressleep/metrics-tpl/internal/model"
	"github.com/Eressleep/metrics-tpl/internal/storage"
	"github.com/gin-gonic/gin"
)

type MetricsHandler struct {
	storage           storage.Storage
	hashKey           string
	usingDB           bool
	dbConnectionError error
}

func NewMetricsHandler(store storage.Storage, hashKey string) *MetricsHandler {
	return &MetricsHandler{
		storage: store,
		hashKey: hashKey,
		usingDB: false,
	}
}

func (h *MetricsHandler) SetUsingDB(usingDB bool) {
	h.usingDB = usingDB
}

func (h *MetricsHandler) IsUsingDB() bool {
	return h.usingDB
}

func (h *MetricsHandler) SetDBConnectionError(err error) {
	h.dbConnectionError = err
}

func (h *MetricsHandler) GetHashKey() string {
	return h.hashKey
}

// Update handles POST /update/:type/:name/:value
func (h *MetricsHandler) Update(c *gin.Context) {
	metricType := c.Param("type")
	metricName := c.Param("name")
	metricValue := c.Param("value")

	if metricName == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "metric name is required"})
		return
	}

	switch metricType {
	case model.Gauge:
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid gauge value"})
			return
		}
		if err := h.storage.UpdateGauge(metricName, value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})

	case model.Counter:
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid counter value"})
			return
		}
		if err := h.storage.UpdateCounter(metricName, value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid metric type"})
	}
}

// UpdateJSON handles POST /update/ (JSON)
func (h *MetricsHandler) UpdateJSON(c *gin.Context) {
	var metric model.Metrics
	if err := json.NewDecoder(c.Request.Body).Decode(&metric); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	if metric.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "metric name is required"})
		return
	}

	switch metric.MType {
	case model.Gauge:
		if metric.Value == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "gauge value is required"})
			return
		}
		if err := h.storage.UpdateGauge(metric.ID, *metric.Value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		updatedValue, _ := h.storage.GetGauge(metric.ID)
		c.JSON(http.StatusOK, model.Metrics{
			ID:    metric.ID,
			MType: model.Gauge,
			Value: &updatedValue,
		})

	case model.Counter:
		if metric.Delta == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "counter delta is required"})
			return
		}
		if err := h.storage.UpdateCounter(metric.ID, *metric.Delta); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		updatedValue, _ := h.storage.GetCounter(metric.ID)
		c.JSON(http.StatusOK, model.Metrics{
			ID:    metric.ID,
			MType: model.Counter,
			Delta: &updatedValue,
		})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid metric type"})
	}
}

// UpdateBatch handles POST /updates/ (batch)
func (h *MetricsHandler) UpdateBatch(c *gin.Context) {
	var metrics []model.Metrics
	if err := json.NewDecoder(c.Request.Body).Decode(&metrics); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	storageMetrics := make([]storage.Metrics, len(metrics))
	for i, m := range metrics {
		storageMetrics[i] = storage.Metrics{
			ID:    m.ID,
			MType: m.MType,
			Delta: m.Delta,
			Value: m.Value,
		}
	}

	if err := h.storage.BatchUpdate(c.Request.Context(), storageMetrics); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetValue handles GET /value/:type/:name
func (h *MetricsHandler) GetValue(c *gin.Context) {
	metricType := c.Param("type")
	metricName := c.Param("name")

	if metricName == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "metric name is required"})
		return
	}

	switch metricType {
	case model.Gauge:
		value, err := h.storage.GetGauge(metricName)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.String(http.StatusOK, strconv.FormatFloat(value, 'g', -1, 64))

	case model.Counter:
		value, err := h.storage.GetCounter(metricName)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.String(http.StatusOK, strconv.FormatInt(value, 10))

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid metric type"})
	}
}

// GetValueJSON handles POST /value/ (JSON)
func (h *MetricsHandler) GetValueJSON(c *gin.Context) {
	var metric model.Metrics
	if err := json.NewDecoder(c.Request.Body).Decode(&metric); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	if metric.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "metric name is required"})
		return
	}

	switch metric.MType {
	case model.Gauge:
		value, err := h.storage.GetGauge(metric.ID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, model.Metrics{
			ID:    metric.ID,
			MType: model.Gauge,
			Value: &value,
		})

	case model.Counter:
		value, err := h.storage.GetCounter(metric.ID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, model.Metrics{
			ID:    metric.ID,
			MType: model.Counter,
			Delta: &value,
		})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid metric type"})
	}
}

// GetAllMetrics handles GET /
func (h *MetricsHandler) GetAllMetrics(c *gin.Context) {
	gauges := h.storage.GetAllGauges()
	counters := h.storage.GetAllCounters()

	html := "<html><body><h1>Metrics</h1>"
	html += "<h2>Gauges</h2><ul>"
	for name, value := range gauges {
		html += fmt.Sprintf("<li>%s: %f</li>", template.HTMLEscapeString(name), value)
	}
	html += "</ul>"
	html += "<h2>Counters</h2><ul>"
	for name, value := range counters {
		html += fmt.Sprintf("<li>%s: %d</li>", template.HTMLEscapeString(name), value)
	}
	html += "</ul></body></html>"

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// Ping handles GET /ping
func (h *MetricsHandler) Ping(c *gin.Context) {
	if h.dbConnectionError != nil {
		errStr := h.dbConnectionError.Error()
		if strings.Contains(errStr, "invalid") || strings.Contains(errStr, "unknown") {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("database connection failed: %v", h.dbConnectionError),
			})
			return
		}
	}

	if h.usingDB {
		if pingable, ok := h.storage.(interface{ Ping() error }); ok {
			if err := pingable.Ping(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": fmt.Sprintf("database ping failed: %v", err),
				})
				return
			}
		}
	}

	c.String(http.StatusOK, "pong")
}

// PingDB handles GET /ping-db
func (h *MetricsHandler) PingDB(c *gin.Context) {
	if !h.usingDB {
		if h.dbConnectionError != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("database connection failed: %v", h.dbConnectionError),
			})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "database is not configured",
		})
		return
	}

	if pingable, ok := h.storage.(interface{ Ping() error }); ok {
		if err := pingable.Ping(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("database ping failed: %v", err),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	c.JSON(http.StatusServiceUnavailable, gin.H{
		"error": "ping not supported for this storage type",
	})
}
