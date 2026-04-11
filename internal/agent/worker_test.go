// internal/gent/worker_test.go
package agent

import (
	"github.com/Eressleep/metrics-tpl/internal/model"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewWorkerPool(t *testing.T) {
	tests := []struct {
		name       string
		workers    int
		serverAddr string
		hashKey    string
		want       int
	}{
		{
			name:       "positive workers",
			workers:    5,
			serverAddr: "localhost:8080",
			hashKey:    "test",
			want:       5,
		},
		{
			name:       "zero workers",
			workers:    0,
			serverAddr: "localhost:8080",
			hashKey:    "",
			want:       1,
		},
		{
			name:       "negative workers",
			workers:    -1,
			serverAddr: "localhost:8080",
			hashKey:    "",
			want:       1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewWorkerPool(tt.workers, tt.serverAddr, tt.hashKey)
			assert.Equal(t, tt.want, pool.workers)
			assert.NotNil(t, pool.jobs)
			assert.NotNil(t, pool.client)
		})
	}
}

func TestWorkerPool_Submit(t *testing.T) {
	pool := NewWorkerPool(1, "localhost:8080", "")
	pool.Start()
	defer pool.Stop()

	metric := model.Metrics{
		ID:    "test",
		MType: model.Gauge,
		Value: floatPtr(123.45),
	}

	assert.True(t, pool.Submit(metric))

	for i := 0; i < 100; i++ {
		pool.Submit(metric)
	}

	assert.False(t, pool.Submit(metric))
}

func TestWorkerPool_Stop(t *testing.T) {
	pool := NewWorkerPool(2, "localhost:8080", "")
	pool.Start()

	metric := model.Metrics{
		ID:    "test",
		MType: model.Gauge,
		Value: floatPtr(123.45),
	}
	pool.Submit(metric)

	pool.Stop()

	pool.Stop()

}

func TestWorkerPool_Concurrent(t *testing.T) {
	pool := NewWorkerPool(5, "localhost:8080", "")
	pool.Start()
	defer pool.Stop()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			metric := model.Metrics{
				ID:    "test",
				MType: model.Counter,
				Delta: intPtr(int64(id)),
			}
			for j := 0; j < 10; j++ {
				pool.Submit(metric)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func floatPtr(f float64) *float64 {
	return &f
}

func intPtr(i int64) *int64 {
	return &i
}
