package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxAttempts != 3 {
		t.Errorf("Expected MaxAttempts=3, got %d", cfg.MaxAttempts)
	}
	if cfg.BaseDelay != 1*time.Second {
		t.Errorf("Expected BaseDelay=1s, got %v", cfg.BaseDelay)
	}
	if cfg.MaxDelay != 5*time.Second {
		t.Errorf("Expected MaxDelay=5s, got %v", cfg.MaxDelay)
	}
}

func TestDo_Success(t *testing.T) {
	ctx := context.Background()
	called := 0

	fn := func() error {
		called++
		return nil
	}

	err := Do(ctx, fn, nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if called != 1 {
		t.Errorf("Expected function called once, got %d", called)
	}
}

func TestDo_NonRetriableError(t *testing.T) {
	ctx := context.Background()
	nonRetriableErr := errors.New("validation error")
	called := 0

	fn := func() error {
		called++
		return nonRetriableErr
	}

	config := &Config{
		MaxAttempts: 3,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    50 * time.Millisecond,
	}

	err := Do(ctx, fn, config)

	if err != nonRetriableErr {
		t.Errorf("Expected original error, got %v", err)
	}
	if called != 1 {
		t.Errorf("Expected function called once, got %d", called)
	}
}

func TestDo_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fn := func() error {
		return errors.New("should not be called")
	}

	err := Do(ctx, fn, nil)

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestDo_DelayCalculation(t *testing.T) {
	ctx := context.Background()
	delays := []time.Duration{}
	start := time.Now()

	fn := func() error {
		if len(delays) == 0 {
			delays = append(delays, 0)
		} else {
			delays = append(delays, time.Since(start))
			start = time.Now()
		}
		return errors.New("error")
	}

	config := &Config{
		MaxAttempts: 4,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    50 * time.Millisecond,
	}

	Do(ctx, fn, config)

	expectedDelays := []time.Duration{0, 10, 20, 40}
	for i := 1; i < len(expectedDelays) && i < len(delays); i++ {
		if delays[i] < expectedDelays[i]-5*time.Millisecond ||
			delays[i] > expectedDelays[i]+15*time.Millisecond {
			t.Errorf("Expected delay ~%v, got %v", expectedDelays[i], delays[i])
		}
	}
}

func TestDo_DelayMaxBound(t *testing.T) {
	ctx := context.Background()
	delays := []time.Duration{}
	start := time.Now()

	fn := func() error {
		if len(delays) == 0 {
			delays = append(delays, 0)
		} else {
			delays = append(delays, time.Since(start))
			start = time.Now()
		}
		return errors.New("error")
	}

	config := &Config{
		MaxAttempts: 10,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    30 * time.Millisecond,
	}

	Do(ctx, fn, config)

	for i := 1; i < len(delays); i++ {
		if delays[i] > 35*time.Millisecond {
			t.Errorf("Delay %v exceeded MaxDelay %v", delays[i], config.MaxDelay)
		}
	}
}

func TestIsRetriable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"connection refused", errors.New("connection refused"), true},
		{"no such host", errors.New("no such host"), true},
		{"network is unreachable", errors.New("network is unreachable"), true},
		{"connection reset", errors.New("connection reset"), true},
		{"broken pipe", errors.New("broken pipe"), true},
		{"EOF", errors.New("EOF"), true},
		{"timeout", errors.New("timeout"), true},
		{"TLS handshake timeout", errors.New("TLS handshake timeout"), true},
		{"i/o timeout", errors.New("i/o timeout"), true},
		{"context deadline exceeded", errors.New("context deadline exceeded"), true},
		{"could not connect to server", errors.New("could not connect to server"), true},
		{"connection to server failed", errors.New("connection to server failed"), true},
		{"server closed the connection", errors.New("server closed the connection"), true},
		{"terminating connection", errors.New("terminating connection"), true},
		{"too many clients", errors.New("too many clients"), true},
		{"failed to read from connection", errors.New("failed to read from connection"), true},
		{"failed to write to connection", errors.New("failed to write to connection"), true},
		{"validation error", errors.New("validation error"), false},
		{"some random error", errors.New("some random error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetriable(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetriable(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}
