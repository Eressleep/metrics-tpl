package storage

import (
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
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if err := fs.MemStorage.UpdateCounter(name, value); err != nil {
		return err
	}

	if fs.storeInterval == 0 {
		return fs.saveToFile()
	}
	return nil
}

func (fs *FileStorage) UpdateGauge(name string, value float64) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if err := fs.MemStorage.UpdateGauge(name, value); err != nil {
		return err
	}

	if fs.storeInterval == 0 {
		return fs.saveToFile()
	}
	return nil
}

func (fs *FileStorage) saveToFile() error {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var metrics []model.Metrics

	for name, value := range fs.gauges {
		val := value
		metrics = append(metrics, model.Metrics{
			ID:    name,
			MType: model.Gauge,
			Value: &val,
		})
	}

	for name, value := range fs.counters {
		val := value
		metrics = append(metrics, model.Metrics{
			ID:    name,
			MType: model.Counter,
			Delta: &val,
		})
	}

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
	close(fs.stopChan)
	fs.wg.Wait()

	return fs.saveToFile()
}
