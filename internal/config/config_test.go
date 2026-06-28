package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadServerConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configData := ServerConfig{
		Address:       "localhost:9090",
		Restore:       boolPtr(false),
		StoreInterval: "30s",
		StoreFile:     "/tmp/test.db",
		DatabaseDSN:   "postgres://user:pass@localhost:5432/db",
		CryptoKey:     "/tmp/key.pem",
		AuditFile:     "/tmp/audit.log",
		AuditURL:      "http://audit.local",
		HashKey:       "test-key",
	}

	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	loaded, err := LoadServerConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loaded.Address != configData.Address {
		t.Errorf("Expected Address %s, got %s", configData.Address, loaded.Address)
	}
	if *loaded.Restore != *configData.Restore {
		t.Errorf("Expected Restore %v, got %v", *configData.Restore, *loaded.Restore)
	}
	if loaded.StoreInterval != configData.StoreInterval {
		t.Errorf("Expected StoreInterval %s, got %s", configData.StoreInterval, loaded.StoreInterval)
	}
	if loaded.StoreFile != configData.StoreFile {
		t.Errorf("Expected StoreFile %s, got %s", configData.StoreFile, loaded.StoreFile)
	}
	if loaded.DatabaseDSN != configData.DatabaseDSN {
		t.Errorf("Expected DatabaseDSN %s, got %s", configData.DatabaseDSN, loaded.DatabaseDSN)
	}
	if loaded.CryptoKey != configData.CryptoKey {
		t.Errorf("Expected CryptoKey %s, got %s", configData.CryptoKey, loaded.CryptoKey)
	}
}

func TestLoadAgentConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configData := AgentConfig{
		Address:        "localhost:9090",
		ReportInterval: "15s",
		PollInterval:   "3s",
		CryptoKey:      "/tmp/key.pem",
		HashKey:        "test-key",
		RateLimit:      5,
	}

	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	loaded, err := LoadAgentConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loaded.Address != configData.Address {
		t.Errorf("Expected Address %s, got %s", configData.Address, loaded.Address)
	}
	if loaded.ReportInterval != configData.ReportInterval {
		t.Errorf("Expected ReportInterval %s, got %s", configData.ReportInterval, loaded.ReportInterval)
	}
	if loaded.PollInterval != configData.PollInterval {
		t.Errorf("Expected PollInterval %s, got %s", configData.PollInterval, loaded.PollInterval)
	}
	if loaded.CryptoKey != configData.CryptoKey {
		t.Errorf("Expected CryptoKey %s, got %s", configData.CryptoKey, loaded.CryptoKey)
	}
	if loaded.HashKey != configData.HashKey {
		t.Errorf("Expected HashKey %s, got %s", configData.HashKey, loaded.HashKey)
	}
	if loaded.RateLimit != configData.RateLimit {
		t.Errorf("Expected RateLimit %d, got %d", configData.RateLimit, loaded.RateLimit)
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	config, err := LoadServerConfig("/path/to/nonexistent.json")
	if err != nil {
		t.Errorf("Expected no error for non-existent file, got %v", err)
	}
	if config != nil {
		t.Errorf("Expected nil config for non-existent file")
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		{"", 0, false},
		{"1s", time.Second, false},
		{"5m", 5 * time.Minute, false},
		{"1h", time.Hour, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		result, err := ParseDuration(tt.input)
		if tt.hasError && err == nil {
			t.Errorf("Expected error for input %q", tt.input)
		}
		if !tt.hasError && err != nil {
			t.Errorf("Unexpected error for input %q: %v", tt.input, err)
		}
		if result != tt.expected {
			t.Errorf("Expected %v, got %v for input %q", tt.expected, result, tt.input)
		}
	}
}

func TestGetBoolOrDefault(t *testing.T) {
	tests := []struct {
		input      *bool
		defaultVal bool
		expected   bool
	}{
		{nil, true, true},
		{nil, false, false},
		{boolPtr(true), false, true},
		{boolPtr(false), true, false},
	}

	for _, tt := range tests {
		result := GetBoolOrDefault(tt.input, tt.defaultVal)
		if result != tt.expected {
			t.Errorf("Expected %v, got %v for input %v", tt.expected, result, tt.input)
		}
	}
}

func boolPtr(b bool) *bool {
	return &b
}
