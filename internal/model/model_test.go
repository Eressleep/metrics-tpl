package model

import (
	"encoding/json"
	"testing"
)

func TestConstants(t *testing.T) {
	if Counter != "counter" {
		t.Errorf("Expected Counter='counter', got '%s'", Counter)
	}
	if Gauge != "gauge" {
		t.Errorf("Expected Gauge='gauge', got '%s'", Gauge)
	}
}

func TestMetrics_JSON_Marshaling(t *testing.T) {
	tests := []struct {
		name     string
		metric   Metrics
		expected string
	}{
		{
			name: "gauge metric with value",
			metric: Metrics{
				ID:    "test_gauge",
				MType: Gauge,
				Value: func() *float64 { v := 123.45; return &v }(),
			},
			expected: `{"id":"test_gauge","type":"gauge","value":123.45}`,
		},
		{
			name: "counter metric with delta",
			metric: Metrics{
				ID:    "test_counter",
				MType: Counter,
				Delta: func() *int64 { v := int64(42); return &v }(),
			},
			expected: `{"id":"test_counter","type":"counter","delta":42}`,
		},
		{
			name: "metric with hash",
			metric: Metrics{
				ID:    "test_hash",
				MType: Gauge,
				Value: func() *float64 { v := 123.45; return &v }(),
				Hash:  "abc123",
			},
			expected: `{"id":"test_hash","type":"gauge","value":123.45,"hash":"abc123"}`,
		},
		{
			name: "gauge with zero value",
			metric: Metrics{
				ID:    "zero_gauge",
				MType: Gauge,
				Value: func() *float64 { v := 0.0; return &v }(),
			},
			expected: `{"id":"zero_gauge","type":"gauge","value":0}`,
		},
		{
			name: "counter with zero delta",
			metric: Metrics{
				ID:    "zero_counter",
				MType: Counter,
				Delta: func() *int64 { v := int64(0); return &v }(),
			},
			expected: `{"id":"zero_counter","type":"counter","delta":0}`,
		},
		{
			name: "omitempty fields",
			metric: Metrics{
				ID:    "test",
				MType: Gauge,
			},
			expected: `{"id":"test","type":"gauge"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.metric)
			if err != nil {
				t.Fatal(err)
			}

			if string(data) != tt.expected {
				t.Errorf("JSON mismatch:\nExpected: %s\nGot:      %s", tt.expected, string(data))
			}

			var unmarshaled Metrics
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Fatal(err)
			}

			if unmarshaled.ID != tt.metric.ID {
				t.Errorf("ID mismatch: expected %s, got %s", tt.metric.ID, unmarshaled.ID)
			}
			if unmarshaled.MType != tt.metric.MType {
				t.Errorf("MType mismatch: expected %s, got %s", tt.metric.MType, unmarshaled.MType)
			}

			if tt.metric.Delta != nil {
				if unmarshaled.Delta == nil {
					t.Error("Delta is nil after unmarshal")
				} else if *unmarshaled.Delta != *tt.metric.Delta {
					t.Errorf("Delta mismatch: expected %d, got %d", *tt.metric.Delta, *unmarshaled.Delta)
				}
			}

			if tt.metric.Value != nil {
				if unmarshaled.Value == nil {
					t.Error("Value is nil after unmarshal")
				} else if *unmarshaled.Value != *tt.metric.Value {
					t.Errorf("Value mismatch: expected %f, got %f", *tt.metric.Value, *unmarshaled.Value)
				}
			}

			if unmarshaled.Hash != tt.metric.Hash {
				t.Errorf("Hash mismatch: expected %s, got %s", tt.metric.Hash, unmarshaled.Hash)
			}
		})
	}
}

func TestMetrics_PointerSemantics(t *testing.T) {
	metric := Metrics{
		ID:    "test",
		MType: Gauge,
	}

	data, err := json.Marshal(metric)
	if err != nil {
		t.Fatal(err)
	}

	var obj map[string]interface{}
	err = json.Unmarshal(data, &obj)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := obj["delta"]; ok {
		t.Error("delta field should be omitted")
	}
	if _, ok := obj["value"]; ok {
		t.Error("value field should be omitted")
	}
	if _, ok := obj["hash"]; ok {
		t.Error("hash field should be omitted")
	}
}

func TestMetrics_ZeroValues(t *testing.T) {
	zeroFloat := 0.0
	zeroInt := int64(0)

	tests := []struct {
		name     string
		metric   Metrics
		checkNil bool
	}{
		{
			name: "zero gauge",
			metric: Metrics{
				ID:    "zero_gauge",
				MType: Gauge,
				Value: &zeroFloat,
			},
			checkNil: false,
		},
		{
			name: "zero counter",
			metric: Metrics{
				ID:    "zero_counter",
				MType: Counter,
				Delta: &zeroInt,
			},
			checkNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.metric)
			if err != nil {
				t.Fatal(err)
			}

			var unmarshaled Metrics
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Fatal(err)
			}

			if tt.metric.Value != nil {
				if unmarshaled.Value == nil {
					t.Error("Value should not be nil for zero value")
				} else if *unmarshaled.Value != 0 {
					t.Errorf("Expected 0, got %f", *unmarshaled.Value)
				}
			}

			if tt.metric.Delta != nil {
				if unmarshaled.Delta == nil {
					t.Error("Delta should not be nil for zero value")
				} else if *unmarshaled.Delta != 0 {
					t.Errorf("Expected 0, got %d", *unmarshaled.Delta)
				}
			}
		})
	}
}

func TestMetrics_EdgeCases(t *testing.T) {
	t.Run("empty struct", func(t *testing.T) {
		metric := Metrics{}
		data, err := json.Marshal(metric)
		if err != nil {
			t.Fatal(err)
		}
		if len(data) < 10 {
			t.Errorf("Unexpected JSON: %s", string(data))
		}
	})

	t.Run("very long strings", func(t *testing.T) {
		longID := string(make([]byte, 1000))
		metric := Metrics{
			ID:    longID,
			MType: Gauge,
			Value: func() *float64 { v := 123.45; return &v }(),
		}
		data, err := json.Marshal(metric)
		if err != nil {
			t.Fatal(err)
		}
		if len(data) < 1000 {
			t.Error("Long ID not properly serialized")
		}
	})

	t.Run("special characters", func(t *testing.T) {
		metric := Metrics{
			ID:    "test\"with\"quotes",
			MType: Gauge,
			Value: func() *float64 { v := 123.45; return &v }(),
			Hash:  "hash\nwith\nnewlines",
		}
		data, err := json.Marshal(metric)
		if err != nil {
			t.Fatal(err)
		}

		var unmarshaled Metrics
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatal(err)
		}

		if unmarshaled.ID != "test\"with\"quotes" {
			t.Errorf("Special characters not preserved: %s", unmarshaled.ID)
		}
		if unmarshaled.Hash != "hash\nwith\nnewlines" {
			t.Errorf("Newlines not preserved: %s", unmarshaled.Hash)
		}
	})
}

func BenchmarkMetrics_Marshal(b *testing.B) {
	metric := Metrics{
		ID:    "bench_metric",
		MType: Gauge,
		Value: func() *float64 { v := 123.45; return &v }(),
		Hash:  "benchhash",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(metric)
	}
}

func BenchmarkMetrics_Unmarshal(b *testing.B) {
	data := []byte(`{"id":"bench_metric","type":"gauge","value":123.45,"hash":"benchhash"}`)
	var metric Metrics

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Unmarshal(data, &metric)
	}
}
