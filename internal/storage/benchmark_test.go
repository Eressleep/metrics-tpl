package storage

import (
	"context"
	"fmt"
	"testing"
)

func BenchmarkMemStorageUpdateCounter(b *testing.B) {
	store := NewMemStorage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.UpdateCounter(fmt.Sprintf("counter_%d", i%100), 1)
	}
}

func BenchmarkMemStorageUpdateGauge(b *testing.B) {
	store := NewMemStorage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.UpdateGauge(fmt.Sprintf("gauge_%d", i%100), float64(i))
	}
}

func BenchmarkMemStorageBatchUpdate(b *testing.B) {
	store := NewMemStorage()
	ctx := context.Background()

	metrics := make([]Metrics, 10)
	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			v := float64(i)
			metrics[i] = Metrics{
				ID:    fmt.Sprintf("gauge_%d", i),
				MType: "gauge",
				Value: &v,
			}
		} else {
			d := int64(i)
			metrics[i] = Metrics{
				ID:    fmt.Sprintf("counter_%d", i),
				MType: "counter",
				Delta: &d,
			}
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.BatchUpdate(ctx, metrics)
	}
}

func BenchmarkMemStorageGetAllGauges(b *testing.B) {
	store := NewMemStorage()

	for i := 0; i < 100; i++ {
		store.UpdateGauge(fmt.Sprintf("gauge_%d", i), float64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.GetAllGauges()
	}
}
