package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Eressleep/metrics-tpl/internal/audit"
	"github.com/Eressleep/metrics-tpl/internal/model"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func setupTestRouter(auditor *audit.Auditor) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	logger := zap.NewNop()

	router.Use(AuditMiddleware(auditor, logger))

	router.POST("/update", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.POST("/updates", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.POST("/update/:type/:name/:value", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.GET("/value/:type/:name", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"value": "test"})
	})

	return router
}

func TestAuditMiddlewareDisabled(t *testing.T) {
	router := setupTestRouter(nil)

	req := httptest.NewRequest(http.MethodPost, "/update", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestExtractMetricNamesURLUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/123.45", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	logger := zap.NewNop()
	metrics := extractMetricNames(c, nil, logger)

	if len(metrics) != 1 || metrics[0] != "Alloc" {
		t.Errorf("expected ['Alloc'], got %v", metrics)
	}
}

func TestExtractMetricNamesJSONUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	metric := model.Metrics{
		ID:    "TestMetric",
		MType: model.Gauge,
	}
	body, _ := json.Marshal(metric)

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	logger := zap.NewNop()
	metrics := extractMetricNames(c, body, logger)

	if len(metrics) != 1 || metrics[0] != "TestMetric" {
		t.Errorf("expected ['TestMetric'], got %v", metrics)
	}
}

func TestExtractMetricNamesBatchUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	batch := []model.Metrics{
		{ID: "Metric1", MType: model.Gauge},
		{ID: "Metric2", MType: model.Counter},
		{ID: "Metric3", MType: model.Gauge},
	}
	body, _ := json.Marshal(batch)

	req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	logger := zap.NewNop()
	metrics := extractMetricNames(c, body, logger)

	if len(metrics) != 3 {
		t.Errorf("expected 3 metrics, got %d", len(metrics))
	}
	if metrics[0] != "Metric1" || metrics[1] != "Metric2" || metrics[2] != "Metric3" {
		t.Errorf("unexpected metrics: %v", metrics)
	}
}

func TestAuditMiddlewareGETRequest(t *testing.T) {
	logger := zap.NewNop()
	auditor := audit.New("test_audit.log", "", logger)
	defer func() {
		recover()
	}()

	router := setupTestRouter(auditor)

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAuditMiddlewareFailedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	router := gin.New()
	router.Use(AuditMiddleware(audit.New("test_audit.log", "", logger), logger))

	router.POST("/update", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
	})

	metric := model.Metrics{ID: "TestMetric", MType: model.Gauge}
	body, _ := json.Marshal(metric)
	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}
