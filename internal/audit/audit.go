package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Event struct {
	TS        int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

type Auditor struct {
	logger   *zap.Logger
	filePath string
	auditURL string
	client   *http.Client
	mu       sync.Mutex
}

func New(filePath, auditURL string, logger *zap.Logger) *Auditor {
	return &Auditor{
		logger:   logger,
		filePath: filePath,
		auditURL: auditURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (a *Auditor) IsEnabled() bool {
	return a.filePath != "" || a.auditURL != ""
}

func (a *Auditor) LogEvent(event Event) error {
	if !a.IsEnabled() {
		return nil
	}

	var errors []error

	if a.filePath != "" {
		if err := a.logToFile(event); err != nil {
			errors = append(errors, fmt.Errorf("file audit error: %w", err))
			a.logger.Error("Failed to write audit to file", zap.Error(err))
		}
	}

	if a.auditURL != "" {
		if err := a.logToURL(event); err != nil {
			errors = append(errors, fmt.Errorf("url audit error: %w", err))
			a.logger.Error("Failed to send audit to URL", zap.Error(err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("audit errors: %v", errors)
	}

	return nil
}

func (a *Auditor) logToFile(event Event) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal audit event: %w", err)
	}

	file, err := os.OpenFile(a.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open audit file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write to audit file: %w", err)
	}

	return nil
}

func (a *Auditor) logToURL(event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal audit event: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, a.auditURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create audit request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send audit event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("audit server returned status %d", resp.StatusCode)
	}

	return nil
}
