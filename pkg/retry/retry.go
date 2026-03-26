package retry

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Config struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

func DefaultConfig() *Config {
	return &Config{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Second,
		MaxDelay:    5 * time.Second,
	}
}

func Do(ctx context.Context, fn func() error, config *Config) error {
	if ctx == nil {
		ctx = context.TODO()
	}

	if config == nil {
		config = DefaultConfig()
	}

	var lastErr error

	for attempt := 0; attempt <= config.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		if attempt == config.MaxAttempts {
			break
		}

		if !IsRetriable(err) {
			return err
		}

		delay := config.BaseDelay * time.Duration(1<<attempt)
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", config.MaxAttempts+1, lastErr)
}

func IsRetriable(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	if isNetworkError(errStr) {
		return true
	}

	if isPostgresConnectionError(errStr) {
		return true
	}

	return false
}

func isNetworkError(errStr string) bool {
	networkErrors := []string{
		"connection refused",
		"no such host",
		"network is unreachable",
		"connection reset",
		"broken pipe",
		"EOF",
		"timeout",
		"TLS handshake timeout",
		"i/o timeout",
		"context deadline exceeded",
	}

	for _, pattern := range networkErrors {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

func isPostgresConnectionError(errStr string) bool {
	postgresErrors := []string{
		"could not connect to server",
		"connection to server",
		"server closed the connection",
		"terminating connection",
		"too many clients",
		"failed to read from connection",
		"failed to write to connection",
	}

	for _, pattern := range postgresErrors {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}
