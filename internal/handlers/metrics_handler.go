package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Eressleep/metrics-tpl/internal/storage"
)

type MetricsHandler struct {
	storage storage.Storage
	*ResponseWriterHelper
}

func NewMetricsHandler(storage storage.Storage) *MetricsHandler {
	return &MetricsHandler{
		storage:              storage,
		ResponseWriterHelper: &ResponseWriterHelper{},
	}
}

func (h *MetricsHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.Error(w, http.StatusMethodNotAllowed, "Only POST allowed")
		return
	}

	if err := validateContentType(r); err != nil {
		h.Error(w, http.StatusNotFound, err.Error())
		return
	}

	metricType, metricName, metricValue, err := parseMetricPath(r.URL.Path)
	if err != nil {
		h.Error(w, http.StatusNotFound, err.Error())
		return
	}

	switch metricType {
	case "counter":
		err = h.handleCounter(metricName, metricValue)
	case "gauge":
		err = h.handleGauge(metricName, metricValue)
	default:
		h.Error(w, http.StatusBadRequest, "Unknown metric type")
		return
	}

	if err != nil {
		h.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

func validateContentType(r *http.Request) error {
	contentType := r.Header.Get("Content-Type")
	if contentType != "" && contentType != "text/plain" {
		return errors.New("invalid content type, expected text/plain")
	}
	return nil
}

func parseMetricPath(urlPath string) (metricType, metricName, metricValue string, err error) {
	path := strings.Split(urlPath, "/")

	if len(path) < 4 {
		return "", "", "", errors.New("invalid URL format: missing metric type or name")
	}

	metricType = path[2]
	metricName = path[3]

	if metricName == "" {
		return "", "", "", errors.New("metric name is required")
	}

	if len(path) < 5 {
		return "", "", "", errors.New("metric value is required")
	}

	metricValue = path[4]
	if metricValue == "" {
		return "", "", "", errors.New("metric value cannot be empty")
	}

	return metricType, metricName, metricValue, nil
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
