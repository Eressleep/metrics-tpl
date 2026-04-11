package storage

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/model"
	"go.uber.org/zap"
)

type FileStorage struct {
	*MemStorage
	filePath      string
	storeInterval time.Duration
	logger        *zap.Logger
	stopChan      chan struct{}
	wg            sync.WaitGroup
	mu            sync.RWMutex
	dataMu        sync.RWMutex
}

func NewFileStorage(filePath string, storeInterval time.Duration, restore bool, logger *zap.Logger) (*FileStorage, error) {
	fs := &FileStorage{
		MemStorage:    NewMemStorage(),
		filePath:      filePath,
		storeInterval: storeInterval,
		logger:        logger,
		stopChan:      make(chan struct{}),
	}

	if restore {
		if err := fs.loadFromFile(); err != nil {
			logger.Warn("Failed to load metrics from file", zap.Error(err))
		}
	}

	if storeInterval > 0 {
		fs.startPeriodicSave()
	}

	return fs, nil
}

func (fs *FileStorage) startPeriodicSave() {
	fs.wg.Add(1)
	go func() {
		defer fs.wg.Done()
		ticker := time.NewTicker(fs.storeInterval)
		defer ticker.Stop()

		for {
			select {
			case <-fs.stopChan:
				return
			case <-ticker.C:
				if err := fs.saveToFile(); err != nil {
					fs.logger.Error("Failed to save metrics to file", zap.Error(err))
				}
			}
		}
	}()
}

func (fs *FileStorage) UpdateCounter(name string, value int64) error {
	fs.dataMu.Lock()
	err := fs.MemStorage.UpdateCounter(name, value)
	fs.dataMu.Unlock()

	if err != nil {
		return err
	}

	if fs.storeInterval == 0 {
		return fs.saveToFile()
	}
	return nil
}

func (fs *FileStorage) UpdateGauge(name string, value float64) error {
	fs.dataMu.Lock()
	err := fs.MemStorage.UpdateGauge(name, value)
	fs.dataMu.Unlock()

	if err != nil {
		return err
	}

	if fs.storeInterval == 0 {
		return fs.saveToFile()
	}
	return nil
}

func (fs *FileStorage) BatchUpdate(ctx context.Context, metrics []Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	fs.dataMu.Lock()
	defer fs.dataMu.Unlock()

	for _, m := range metrics {
		switch m.MType {
		case "counter":
			if m.Delta != nil {
				if err := fs.MemStorage.UpdateCounter(m.ID, *m.Delta); err != nil {
					return err
				}
			}
		case "gauge":
			if m.Value != nil {
				if err := fs.MemStorage.UpdateGauge(m.ID, *m.Value); err != nil {
					return err
				}
			}
		}
	}

	if fs.storeInterval == 0 {
		return fs.saveToFile()
	}

	return nil
}

func (fs *FileStorage) GetCounter(name string) (int64, error) {
	fs.dataMu.RLock()
	defer fs.dataMu.RUnlock()
	return fs.MemStorage.GetCounter(name)
}

func (fs *FileStorage) GetGauge(name string) (float64, error) {
	fs.dataMu.RLock()
	defer fs.dataMu.RUnlock()
	return fs.MemStorage.GetGauge(name)
}

func (fs *FileStorage) GetAllGauges() map[string]float64 {
	fs.dataMu.RLock()
	defer fs.dataMu.RUnlock()
	return fs.MemStorage.GetAllGauges()
}

func (fs *FileStorage) GetAllCounters() map[string]int64 {
	fs.dataMu.RLock()
	defer fs.dataMu.RUnlock()
	return fs.MemStorage.GetAllCounters()
}

func (fs *FileStorage) saveToFile() error {
	fs.dataMu.RLock()
	gauges := fs.MemStorage.GetAllGauges()
	counters := fs.MemStorage.GetAllCounters()
	fs.dataMu.RUnlock()

	var metrics []model.Metrics

	for name, value := range gauges {
		val := value
		metrics = append(metrics, model.Metrics{
			ID:    name,
			MType: model.Gauge,
			Value: &val,
		})
	}

	for name, value := range counters {
		val := value
		metrics = append(metrics, model.Metrics{
			ID:    name,
			MType: model.Counter,
			Delta: &val,
		})
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	tempFile := fs.filePath + ".tmp"
	file, err := os.Create(tempFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(metrics); err != nil {
		return err
	}

	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	return os.Rename(tempFile, fs.filePath)
}

func (fs *FileStorage) loadFromFile() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	file, err := os.Open(fs.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	var metrics []model.Metrics
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&metrics); err != nil {
		return err
	}

	fs.dataMu.Lock()
	defer fs.dataMu.Unlock()

	fs.gauges = make(map[string]float64)
	fs.counters = make(map[string]int64)

	for _, metric := range metrics {
		switch metric.MType {
		case model.Gauge:
			if metric.Value != nil {
				fs.gauges[metric.ID] = *metric.Value
			}
		case model.Counter:
			if metric.Delta != nil {
				fs.counters[metric.ID] = *metric.Delta
			}
		}
	}

	return nil
}

func (fs *FileStorage) Stop() error {
	select {
	case <-fs.stopChan:
	default:
		close(fs.stopChan)
	}

	fs.wg.Wait()
	return fs.saveToFile()
}
