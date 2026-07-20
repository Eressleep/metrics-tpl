package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ServerConfig представляет конфигурацию сервера из JSON файла
type ServerConfig struct {
	Address       string `json:"address"`
	Restore       *bool  `json:"restore"`
	StoreInterval string `json:"store_interval"`
	StoreFile     string `json:"store_file"`
	DatabaseDSN   string `json:"database_dsn"`
	CryptoKey     string `json:"crypto_key"`
	AuditFile     string `json:"audit_file"`
	AuditURL      string `json:"audit_url"`
	HashKey       string `json:"hash_key"`
	TrustedSubnet string `json:"trusted_subnet"`
}

// AgentConfig представляет конфигурацию агента из JSON файла
type AgentConfig struct {
	Address        string `json:"address"`
	ReportInterval string `json:"report_interval"`
	PollInterval   string `json:"poll_interval"`
	CryptoKey      string `json:"crypto_key"`
	HashKey        string `json:"hash_key"`
	RateLimit      int    `json:"rate_limit"`
}

// LoadServerConfig загружает конфигурацию сервера из JSON файла
func LoadServerConfig(path string) (*ServerConfig, error) {
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ServerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// LoadAgentConfig загружает конфигурацию агента из JSON файла
func LoadAgentConfig(path string) (*AgentConfig, error) {
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config AgentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// ParseDuration парсит длительность из строки
func ParseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}
	return time.ParseDuration(s)
}

// GetBoolOrDefault возвращает значение bool или значение по умолчанию, если nil
func GetBoolOrDefault(b *bool, defaultValue bool) bool {
	if b == nil {
		return defaultValue
	}
	return *b
}
