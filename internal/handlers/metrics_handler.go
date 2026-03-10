package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/Eressleep/metrics-tpl/internal/storage"
	"github.com/gin-gonic/gin"
)

type MetricsHandler struct {
	storage storage.Storage
}

func NewMetricsHandler(storage storage.Storage) *MetricsHandler {
	return &MetricsHandler{
		storage: storage,
	}
}

func (h *MetricsHandler) Update(c *gin.Context) {
	metricType := c.Param("type")
	metricName := c.Param("name")
	metricValue := c.Param("value")

	if metricName == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "metric name is required"})
		return
	}

	if metricValue == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "metric value is required"})
		return
	}

	contentType := c.GetHeader("Content-Type")
	if contentType != "" && contentType != "text/plain" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid content type, expected text/plain"})
		return
	}

	switch metricType {
	case "counter":
		if err := h.handleCounter(metricName, metricValue); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	case "gauge":
		if err := h.handleGauge(metricName, metricValue); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown metric type"})
		return
	}

	c.Status(http.StatusOK)
}

func (h *MetricsHandler) GetValue(c *gin.Context) {
	metricType := c.Param("type")
	metricName := c.Param("name")

	if metricName == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "metric name is required"})
		return
	}

	switch metricType {
	case "counter":
		value, err := h.storage.GetCounter(metricName)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("counter %s not found", metricName)})
			return
		}
		c.String(http.StatusOK, "%d", value)

	case "gauge":
		value, err := h.storage.GetGauge(metricName)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("gauge %s not found", metricName)})
			return
		}
		c.String(http.StatusOK, "%g", value)

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown metric type"})
		return
	}
}

func (h *MetricsHandler) GetAllMetrics(c *gin.Context) {
	// Получаем все метрики
	gauges := h.storage.GetAllGauges()
	counters := h.storage.GetAllCounters()

	// Используем HTML шаблон Gin
	c.HTML(http.StatusOK, "metrics.tmpl", gin.H{
		"Gauges":   gauges,
		"Counters": counters,
	})
}

func (h *MetricsHandler) handleCounter(name, valueStr string) error {
	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid counter value: %w", err)
	}
	return h.storage.UpdateCounter(name, value)
}

func (h *MetricsHandler) handleGauge(name, valueStr string) error {
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return fmt.Errorf("invalid gauge value: %w", err)
	}
	return h.storage.UpdateGauge(name, value)
}
