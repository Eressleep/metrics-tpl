package storage

import (
	"testing"

	"go.uber.org/zap"
)

func TestNewDBStorage(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	_, err := NewDBStorage("invalid_dsn", logger)
	if err == nil {
		t.Error("Expected error with invalid DSN")
	}

	_, err = NewDBStorage("", logger)
	if err == nil {
		t.Error("Expected error with empty DSN")
	}
}

func TestDBStorageInterface(t *testing.T) {
	var _ Storage = (*DBStorage)(nil)
}
