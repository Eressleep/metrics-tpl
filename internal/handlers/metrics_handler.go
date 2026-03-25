package handlers

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"

	"github.com/Eressleep/metrics-tpl/internal/model"
	"github.com/Eressleep/metrics-tpl/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/mailru/easyjson"
)

func isValidMetricName(name string) bool {
	if len(name) == 0 || len(name) > 100 {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]+$`, name)
	return matched
}

type MetricsHandler struct {
	storage storage.Storage
}

func NewMetricsHandler(storage storage.Storage) *MetricsHandler {
	return &MetricsHandler{
		storage: storage,
	}
}

func (h *MetricsHandler) UpdateJSON(c *gin.Context) {
	if c.GetHeader("Content-Type") != "application/json" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content-Type must be application/json"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body: " + err.Error()})
		return
	}
	defer c.Request.Body.Close()

	var metric model.Metrics
	if err := easyjson.Unmarshal(body, &metric); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON format: " + err.Error()})
		return
	}

	if metric.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "metric ID is required"})
		return
	}

	if !isValidMetricName(metric.ID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid metric name format"})
		return
	}

	if metric.MType != model.Counter && metric.MType != model.Gauge {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unknown metric type: %s", metric.MType)})
		return
	}

	switch metric.MType {
	case model.Counter:
		if metric.Delta == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "delta is required for counter metric"})
			return
		}
		if err := h.storage.UpdateCounter(metric.ID, *metric.Delta); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if value, err := h.storage.GetCounter(metric.ID); err == nil {
			metric.Delta = &value
		}

	case model.Gauge:
		if metric.Value == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "value is required for gauge metric"})
			return
		}
		if err := h.storage.UpdateGauge(metric.ID, *metric.Value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if value, err := h.storage.GetGauge(metric.ID); err == nil {
			metric.Value = &value
		}
	}

	responseData, err := easyjson.Marshal(&metric)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode response"})
		return
	}

	c.Data(http.StatusOK, "application/json", responseData)
}

func (h *MetricsHandler) GetValueJSON(c *gin.Context) {
	if c.GetHeader("Content-Type") != "application/json" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content-Type must be application/json"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body: " + err.Error()})
		return
	}
	defer c.Request.Body.Close()

	var request model.Metrics
	if err := easyjson.Unmarshal(body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON format: " + err.Error()})
		return
	}

	if request.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "metric ID is required"})
		return
	}

	if !isValidMetricName(request.ID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid metric name format"})
		return
	}

	if request.MType != model.Counter && request.MType != model.Gauge {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unknown metric type: %s", request.MType)})
		return
	}

	response := model.Metrics{
		ID:    request.ID,
		MType: request.MType,
	}

	switch request.MType {
	case model.Counter:
		value, err := h.storage.GetCounter(request.ID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("counter %s not found", request.ID)})
			return
		}
		response.Delta = &value

	case model.Gauge:
		value, err := h.storage.GetGauge(request.ID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("gauge %s not found", request.ID)})
			return
		}
		response.Value = &value
	}

	responseData, err := easyjson.Marshal(&response)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode response"})
		return
	}

	c.Data(http.StatusOK, "application/json", responseData)
}

func (h *MetricsHandler) Update(c *gin.Context) {
	metricType := c.Param("type")
	metricName := c.Param("name")
	metricValue := c.Param("value")

	if metricName == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "metric name is required"})
		return
	}

	if !isValidMetricName(metricName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid metric name format"})
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
	case model.Counter:
		if err := h.handleCounter(metricName, metricValue); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	case model.Gauge:
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

	if !isValidMetricName(metricName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid metric name format"})
		return
	}

	switch metricType {
	case model.Counter:
		value, err := h.storage.GetCounter(metricName)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("counter %s not found", metricName)})
			return
		}
		c.String(http.StatusOK, "%d", value)

	case model.Gauge:
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
	gauges := h.storage.GetAllGauges()
	counters := h.storage.GetAllCounters()

	html := "<!DOCTYPE html><html><head><title>Metrics</title><style>"
	html += "body{font-family:Arial;margin:20px;background:#f5f5f5}"
	html += "h1{color:#333} h2{color:#666}"
	html += "table{border-collapse:collapse;width:100%;background:white;box-shadow:0 2px 5px rgba(0,0,0,0.1)}"
	html += "th,td{text-align:left;padding:12px;border-bottom:1px solid #ddd}"
	html += "th{background:#4CAF50;color:white}"
	html += "tr:hover{background:#f5f5f5}"
	html += ".counter{background:#e7f3ff}"
	html += ".stats{margin:20px 0;padding:15px;background:white;border-radius:5px;box-shadow:0 2px 5px rgba(0,0,0,0.1)}"
	html += "</style></head><body>"

	html += "<h1>📊 Metrics Dashboard</h1>"

	html += "<div class='stats'>"
	html += fmt.Sprintf("<p><strong>Total Gauges:</strong> %d</p>", len(gauges))
	html += fmt.Sprintf("<p><strong>Total Counters:</strong> %d</p>", len(counters))
	html += fmt.Sprintf("<p><strong>Total Metrics:</strong> %d</p>", len(gauges)+len(counters))
	html += "</div>"

	html += "<h2>📈 Gauge Metrics</h2>"
	if len(gauges) > 0 {
		html += "<table><tr><th>Metric Name</th><th>Value</th></tr>"
		for name, value := range gauges {
			html += fmt.Sprintf("<tr><td>%s</td><td>%g</td></tr>", name, value)
		}
		html += "</table>"
	} else {
		html += "<p>No gauge metrics available</p>"
	}

	html += "<h2>🔢 Counter Metrics</h2>"
	if len(counters) > 0 {
		html += "<table><tr><th>Metric Name</th><th>Value</th></tr>"
		for name, value := range counters {
			html += fmt.Sprintf("<tr class='counter'><td>%s</td><td>%d</td></tr>", name, value)
		}
		html += "</table>"
	} else {
		html += "<p>No counter metrics available</p>"
	}

	html += "</body></html>"

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
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

func (h *MetricsHandler) Ping(c *gin.Context) {
	c.String(http.StatusOK, "pong")
}

func (h *MetricsHandler) PingDB(c *gin.Context) {
	type pinger interface {
		Ping() error
	}

	if pinger, ok := h.storage.(pinger); ok {
		if err := pinger.Ping(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection failed"})
			return
		}
	}
	c.String(http.StatusOK, "pong")
}

func (h *MetricsHandler) PingDB(c *gin.Context) {
	type pinger interface {
		Ping() error
	}

	if pinger, ok := h.storage.(pinger); ok {
		if err := pinger.Ping(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection failed"})
			return
		}
	}
	c.String(http.StatusOK, "pong")
}
