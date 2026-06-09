package audit

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewAuditor(t *testing.T) {
	logger := zap.NewNop()
	auditor := New("test.log", "http://example.com", logger)

	if auditor.filePath != "test.log" {
		t.Errorf("expected filePath 'test.log', got '%s'", auditor.filePath)
	}
	if auditor.auditURL != "http://example.com" {
		t.Errorf("expected auditURL 'http://example.com', got '%s'", auditor.auditURL)
	}
}

func TestIsEnabled(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name     string
		filePath string
		auditURL string
		expected bool
	}{
		{"both empty", "", "", false},
		{"file only", "test.log", "", true},
		{"url only", "", "http://example.com", true},
		{"both filled", "test.log", "http://example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auditor := New(tt.filePath, tt.auditURL, logger)
			if auditor.IsEnabled() != tt.expected {
				t.Errorf("expected IsEnabled() = %v, got %v", tt.expected, auditor.IsEnabled())
			}
		})
	}
}

func TestLogEventToFile(t *testing.T) {
	logger := zap.NewNop()
	tmpFile := "test_audit.log"
	defer os.Remove(tmpFile)

	auditor := New(tmpFile, "", logger)
	event := Event{
		TS:        time.Now().Unix(),
		Metrics:   []string{"Alloc", "Frees"},
		IPAddress: "127.0.0.1",
	}

	err := auditor.LogEvent(event)
	if err != nil {
		t.Fatalf("LogEvent failed: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read audit file: %v", err)
	}

	var readEvent Event
	err = json.Unmarshal(data[:len(data)-1], &readEvent)
	if err != nil {
		t.Fatalf("Failed to unmarshal audit event: %v", err)
	}

	if readEvent.IPAddress != event.IPAddress {
		t.Errorf("expected IP '%s', got '%s'", event.IPAddress, readEvent.IPAddress)
	}
	if len(readEvent.Metrics) != len(event.Metrics) {
		t.Errorf("expected %d metrics, got %d", len(event.Metrics), len(readEvent.Metrics))
	}
}

func TestLogEventToURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		body, _ := io.ReadAll(r.Body)
		var event Event
		json.Unmarshal(body, &event)

		if event.IPAddress != "192.168.1.1" {
			t.Errorf("expected IP '192.168.1.1', got '%s'", event.IPAddress)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := zap.NewNop()
	auditor := New("", server.URL, logger)

	event := Event{
		TS:        time.Now().Unix(),
		Metrics:   []string{"CPUutilization1"},
		IPAddress: "192.168.1.1",
	}

	err := auditor.LogEvent(event)
	if err != nil {
		t.Fatalf("LogEvent failed: %v", err)
	}
}

func TestLogEventURLServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	logger := zap.NewNop()
	auditor := New("", server.URL, logger)

	event := Event{
		TS:        time.Now().Unix(),
		Metrics:   []string{"TestMetric"},
		IPAddress: "127.0.0.1",
	}

	err := auditor.LogEvent(event)
	if err == nil {
		t.Error("expected error for 500 status code")
	}
}

func TestLogEventDisabled(t *testing.T) {
	logger := zap.NewNop()
	auditor := New("", "", logger)

	event := Event{
		TS:        time.Now().Unix(),
		Metrics:   []string{"TestMetric"},
		IPAddress: "127.0.0.1",
	}

	err := auditor.LogEvent(event)
	if err != nil {
		t.Errorf("expected no error for disabled auditor, got: %v", err)
	}
}

func TestMultipleLogEvents(t *testing.T) {
	logger := zap.NewNop()
	tmpFile := "test_multi_audit.log"
	defer os.Remove(tmpFile)

	auditor := New(tmpFile, "", logger)

	events := []Event{
		{TS: time.Now().Unix(), Metrics: []string{"Metric1"}, IPAddress: "127.0.0.1"},
		{TS: time.Now().Unix(), Metrics: []string{"Metric2", "Metric3"}, IPAddress: "127.0.0.2"},
	}

	for _, event := range events {
		if err := auditor.LogEvent(event); err != nil {
			t.Fatalf("LogEvent failed: %v", err)
		}
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read audit file: %v", err)
	}

	lines := 0
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}

	if lines != 2 {
		t.Errorf("expected 2 lines in audit file, got %d", lines)
	}
}
