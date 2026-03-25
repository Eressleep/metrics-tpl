package storage

import (
	"os"
	"testing"

	_ "github.com/lib/pq"
)

func getTestDSN() string {
	return os.Getenv("TEST_DATABASE_DSN")
}

func TestDBStorage_Integration(t *testing.T) {
	dsn := getTestDSN()
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping integration test")
	}

	db, err := NewDBStorage(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.UpdateGauge("test_gauge", 123.45)
	if err != nil {
		t.Fatal(err)
	}

	value, err := db.GetGauge("test_gauge")
	if err != nil {
		t.Fatal(err)
	}
	if value != 123.45 {
		t.Errorf("Expected 123.45, got %f", value)
	}

	err = db.UpdateCounter("test_counter", 42)
	if err != nil {
		t.Fatal(err)
	}

	counter, err := db.GetCounter("test_counter")
	if err != nil {
		t.Fatal(err)
	}
	if counter != 42 {
		t.Errorf("Expected 42, got %d", counter)
	}

	gauges := db.GetAllGauges()
	if len(gauges) == 0 {
		t.Error("GetAllGauges returned empty")
	}

	counters := db.GetAllCounters()
	if len(counters) == 0 {
		t.Error("GetAllCounters returned empty")
	}

	err = db.Ping()
	if err != nil {
		t.Error("Ping failed:", err)
	}
}
