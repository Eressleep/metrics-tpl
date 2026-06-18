package database

import (
	"testing"
)

func TestNewDB_InvalidDSN(t *testing.T) {
	_, err := NewDB("invalid-dsn")
	if err == nil {
		t.Error("Expected error for invalid DSN")
	}
}

func TestNewDB_ConnectionRefused(t *testing.T) {
	_, err := NewDB("postgres://invalid:invalid@localhost:9999/test?sslmode=disable")
	if err == nil {
		t.Error("Expected error for connection refused")
	}
}
