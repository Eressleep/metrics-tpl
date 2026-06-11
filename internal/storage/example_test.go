package storage_test

import (
	"context"
	"fmt"

	"github.com/Eressleep/metrics-tpl/internal/storage"
)

// ExampleMemStorage демонстрирует использование in-memory хранилища метрик.
func ExampleMemStorage() {
	store := storage.NewMemStorage()

	// Обновляем gauge метрику
	store.UpdateGauge("temperature", 23.5)

	// Обновляем counter метрику
	store.UpdateCounter("requests", 1)
	store.UpdateCounter("requests", 2)

	// Получаем значения
	temp, _ := store.GetGauge("temperature")
	reqs, _ := store.GetCounter("requests")

	fmt.Printf("Temperature: %g\n", temp)
	fmt.Printf("Requests: %d\n", reqs)

	// Получаем все метрики
	gauges := store.GetAllGauges()
	counters := store.GetAllCounters()

	fmt.Printf("Total gauges: %d\n", len(gauges))
	fmt.Printf("Total counters: %d\n", len(counters))

	// Output:
	// Temperature: 23.5
	// Requests: 3
	// Total gauges: 1
	// Total counters: 1
}

// ExampleMemStorage_BatchUpdate демонстрирует пакетное обновление метрик.
func ExampleMemStorage_BatchUpdate() {
	store := storage.NewMemStorage()
	ctx := context.Background()

	v1, v2 := 23.5, 65.0
	d1 := int64(100)

	metrics := []storage.Metrics{
		{ID: "temperature", MType: "gauge", Value: &v1},
		{ID: "humidity", MType: "gauge", Value: &v2},
		{ID: "requests", MType: "counter", Delta: &d1},
	}

	store.BatchUpdate(ctx, metrics)

	temp, _ := store.GetGauge("temperature")
	hum, _ := store.GetGauge("humidity")
	reqs, _ := store.GetCounter("requests")

	fmt.Printf("Temperature: %g\n", temp)
	fmt.Printf("Humidity: %g\n", hum)
	fmt.Printf("Requests: %d\n", reqs)

	// Output:
	// Temperature: 23.5
	// Humidity: 65
	// Requests: 100
}
