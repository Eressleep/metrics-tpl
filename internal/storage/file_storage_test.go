package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/model"
	"go.uber.org/zap/zaptest"
)

func setupTestFileStorage(t *testing.T) (*FileStorage, string, func()) {
	tmpDir, err := os.MkdirTemp("", "storage_test_*")
	if err != nil {
		t.Fatal(err)
	}

	filePath := filepath.Join(tmpDir, "metrics.json")
	logger := zaptest.NewLogger(t)

	fs := &FileStorage{
		MemStorage:    NewMemStorage(),
		filePath:      filePath,
		storeInterval: 0,
		logger:        logger,
		stopChan:      make(chan struct{}),
	}

	cleanup := func() {
		fs.Stop()
		os.RemoveAll(tmpDir)
	}

	return fs, filePath, cleanup
}

func TestNewFileStorage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "metrics.json")
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name          string
		filePath      string
		storeInterval time.Duration
		restore       bool
		wantErr       bool
	}{
		{
			name:          "valid config with restore",
			filePath:      filePath,
			storeInterval: 5 * time.Second,
			restore:       true,
			wantErr:       false,
		},
		{
			name:          "valid config without restore",
			filePath:      filePath,
			storeInterval: 5 * time.Second,
			restore:       false,
			wantErr:       false,
		},
		{
			name:          "sync mode",
			filePath:      filePath,
			storeInterval: 0,
			restore:       true,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs, err := NewFileStorage(tt.filePath, tt.storeInterval, tt.restore, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFileStorage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if fs != nil {
				defer fs.Stop()
			}
		})
	}
}

func TestFileStorage_UpdateAndSave(t *testing.T) {
	fs, filePath, cleanup := setupTestFileStorage(t)
	defer cleanup()

	err := fs.UpdateCounter("test_counter", 42)
	if err != nil {
		t.Fatal(err)
	}

	err = fs.UpdateGauge("test_gauge", 123.45)
	if err != nil {
		t.Fatal(err)
	}

	err = fs.saveToFile()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("File was not created")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	var metrics []model.Metrics
	err = json.Unmarshal(data, &metrics)
	if err != nil {
		t.Fatal(err)
	}

	foundCounter := false
	foundGauge := false

	for _, m := range metrics {
		switch m.ID {
		case "test_counter":
			foundCounter = true
			if m.MType != model.Counter {
				t.Errorf("Expected type counter, got %s", m.MType)
			}
			if m.Delta == nil || *m.Delta != 42 {
				t.Errorf("Expected delta 42, got %v", m.Delta)
			}
		case "test_gauge":
			foundGauge = true
			if m.MType != model.Gauge {
				t.Errorf("Expected type gauge, got %s", m.MType)
			}
			if m.Value == nil || *m.Value != 123.45 {
				t.Errorf("Expected value 123.45, got %v", m.Value)
			}
		}
	}

	if !foundCounter {
		t.Error("Counter metric not found in file")
	}
	if !foundGauge {
		t.Error("Gauge metric not found in file")
	}
}

func TestFileStorage_LoadFromFile(t *testing.T) {
	fs, filePath, cleanup := setupTestFileStorage(t)
	defer cleanup()

	testMetrics := []model.Metrics{
		{
			ID:    "counter1",
			MType: model.Counter,
			Delta: func() *int64 { v := int64(100); return &v }(),
		},
		{
			ID:    "gauge1",
			MType: model.Gauge,
			Value: func() *float64 { v := 123.45; return &v }(),
		},
	}

	data, err := json.MarshalIndent(testMetrics, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = fs.loadFromFile()
	if err != nil {
		t.Fatal(err)
	}

	counter, err := fs.GetCounter("counter1")
	if err != nil {
		t.Errorf("Failed to get counter: %v", err)
	}
	if counter != 100 {
		t.Errorf("Expected counter 100, got %d", counter)
	}

	gauge, err := fs.GetGauge("gauge1")
	if err != nil {
		t.Errorf("Failed to get gauge: %v", err)
	}
	if gauge != 123.45 {
		t.Errorf("Expected gauge 123.45, got %f", gauge)
	}
}

func TestFileStorage_LoadFromNonExistentFile(t *testing.T) {
	fs, _, cleanup := setupTestFileStorage(t)
	defer cleanup()

	err := fs.loadFromFile()
	if err != nil {
		t.Errorf("Expected no error for non-existent file, got %v", err)
	}
}

func TestFileStorage_SyncMode(t *testing.T) {
	fs, filePath, cleanup := setupTestFileStorage(t)
	defer cleanup()

	fs.storeInterval = 0

	err := fs.UpdateCounter("sync_counter", 42)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("File was not created in sync mode")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	var metrics []model.Metrics
	err = json.Unmarshal(data, &metrics)
	if err != nil {
		t.Fatal(err)
	}

	if len(metrics) == 0 {
		t.Error("No metrics found in file")
	}
}

func TestFileStorage_PeriodicSave(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "metrics.json")
	logger := zaptest.NewLogger(t)

	fs, err := NewFileStorage(filePath, 100*time.Millisecond, false, logger)
	if err != nil {
		t.Fatal(err)
	}
	defer fs.Stop()

	err = fs.UpdateCounter("periodic_counter", 42)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(250 * time.Millisecond)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("File was not created by periodic save")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	var metrics []model.Metrics
	err = json.Unmarshal(data, &metrics)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, m := range metrics {
		if m.ID == "periodic_counter" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Metric not found in periodically saved file")
	}
}

func TestFileStorage_Stop(t *testing.T) {
	fs, filePath, cleanup := setupTestFileStorage(t)
	defer cleanup()

	err := fs.UpdateCounter("stop_counter", 42)
	if err != nil {
		t.Fatal(err)
	}

	err = fs.Stop()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("File was not created on Stop")
	}
}

func TestFileStorage_ConcurrentAccess(t *testing.T) {
	fs, filePath, cleanup := setupTestFileStorage(t)
	defer cleanup()

	fs.storeInterval = 10 * time.Millisecond

	fs.startPeriodicSave()
	defer fs.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				fs.UpdateCounter("counter", int64(id))
				fs.UpdateGauge("gauge", float64(id))

				fs.GetCounter("counter")
				fs.GetGauge("gauge")
				fs.GetAllCounters()
				fs.GetAllGauges()
			}
		}(i)
	}

	wg.Wait()

	err := fs.saveToFile()
	if err != nil {
		t.Fatalf("Failed to save file: %v", err)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("File was not created after concurrent access")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	var metrics []model.Metrics
	err = json.Unmarshal(data, &metrics)
	if err != nil {
		t.Fatal(err)
	}

	if len(metrics) == 0 {
		t.Error("File is empty after concurrent access")
	}
}

func TestFileStorage_ConcurrentWrites(t *testing.T) {
	fs, filePath, cleanup := setupTestFileStorage(t)
	defer cleanup()

	fs.storeInterval = 0

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			name := "counter_" + strconv.Itoa(id)
			for j := 0; j < 10; j++ {
				fs.UpdateCounter(name, 1)
			}
		}(i)
	}

	wg.Wait()

	err := fs.saveToFile()
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	var metrics []model.Metrics
	err = json.Unmarshal(data, &metrics)
	if err != nil {
		t.Fatal(err)
	}

	if len(metrics) != 50 {
		t.Errorf("Expected 50 metrics, got %d", len(metrics))
	}
}

func TestFileStorage_UpdateAfterStop(t *testing.T) {
	fs, _, cleanup := setupTestFileStorage(t)
	defer cleanup()

	err := fs.Stop()
	if err != nil {
		t.Fatal(err)
	}

	err = fs.UpdateCounter("after_stop", 42)
	if err != nil {
		t.Errorf("Should be able to update after stop, got error: %v", err)
	}

	value, err := fs.GetCounter("after_stop")
	if err != nil || value != 42 {
		t.Errorf("Expected counter 42, got %d, error: %v", value, err)
	}
}

func TestFileStorage_FilePermissions(t *testing.T) {
	fs, filePath, cleanup := setupTestFileStorage(t)
	defer cleanup()

	err := fs.UpdateCounter("perm_counter", 42)
	if err != nil {
		t.Fatal(err)
	}

	err = fs.saveToFile()
	if err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatal(err)
	}

	if info.Mode()&0400 == 0 {
		t.Error("File is not readable by owner")
	}
	if info.Mode()&0200 == 0 {
		t.Error("File is not writable by owner")
	}
}

func TestFileStorage_LoadCorruptedFile(t *testing.T) {
	fs, filePath, cleanup := setupTestFileStorage(t)
	defer cleanup()

	corruptedData := []byte("this is not valid json")
	err := os.WriteFile(filePath, corruptedData, 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = fs.loadFromFile()
	if err == nil {
		t.Error("Expected error when loading corrupted file, got nil")
	}
}

func TestFileStorage_RestoreOnStart(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "metrics.json")
	logger := zaptest.NewLogger(t)

	testMetrics := []model.Metrics{
		{
			ID:    "restore_counter",
			MType: model.Counter,
			Delta: func() *int64 { v := int64(42); return &v }(),
		},
	}

	data, err := json.MarshalIndent(testMetrics, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		t.Fatal(err)
	}

	fs, err := NewFileStorage(filePath, 5*time.Second, true, logger)
	if err != nil {
		t.Fatal(err)
	}
	defer fs.Stop()

	value, err := fs.GetCounter("restore_counter")
	if err != nil {
		t.Errorf("Failed to get restored counter: %v", err)
	}
	if value != 42 {
		t.Errorf("Expected restored counter 42, got %d", value)
	}
}

func TestFileStorage_AtomicSave(t *testing.T) {
	fs, filePath, cleanup := setupTestFileStorage(t)
	defer cleanup()

	for i := 0; i < 10; i++ {
		err := fs.UpdateCounter("atomic_counter", int64(i))
		if err != nil {
			t.Fatal(err)
		}
	}

	err := fs.saveToFile()
	if err != nil {
		t.Fatal(err)
	}

	tempFile := filePath + ".tmp"
	if _, err := os.Stat(tempFile); !os.IsNotExist(err) {
		t.Error("Temporary file was not removed")
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Main file does not exist")
	}
}
