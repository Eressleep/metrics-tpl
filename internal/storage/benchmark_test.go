package storage

import (
	"context"
	"strconv"
	"testing"
)

func BenchmarkMemStorageUpdateCounter(b *testing.B) {
	store := NewMemStorage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.UpdateCounter("counter_"+strconv.Itoa(i%100), 1)
	}
}

func BenchmarkMemStorageUpdateGauge(b *testing.B) {
	store := NewMemStorage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.UpdateGauge("gauge_"+strconv.Itoa(i%100), float64(i))
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
				ID:    "gauge_" + strconv.Itoa(i),
				MType: "gauge",
				Value: &v,
			}
		} else {
			d := int64(i)
			metrics[i] = Metrics{
				ID:    "counter_" + strconv.Itoa(i),
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

	for i := 0; i < 30; i++ {
		store.UpdateGauge("gauge_"+strconv.Itoa(i), float64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.GetAllGauges()
	}
}

func BenchmarkMemStorageGetAllCounters(b *testing.B) {
	store := NewMemStorage()

	for i := 0; i < 10; i++ {
		store.UpdateCounter("counter_"+strconv.Itoa(i), int64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.GetAllCounters()
	}
}
