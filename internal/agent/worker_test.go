package agent

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewWorkerPool(t *testing.T) {
	tests := []struct {
		name       string
		workers    int
		serverAddr string
		hashKey    string
		expected   int
	}{
		{
			name:       "normal creation",
			workers:    5,
			serverAddr: "localhost:8080",
			hashKey:    "test-key",
			expected:   5,
		},
		{
			name:       "zero workers defaults to 1",
			workers:    0,
			serverAddr: "localhost:8080",
			hashKey:    "",
			expected:   1,
		},
		{
			name:       "negative workers defaults to 1",
			workers:    -5,
			serverAddr: "localhost:8080",
			hashKey:    "key",
			expected:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewWorkerPool(tt.workers, tt.serverAddr, tt.hashKey)

			if pool.workers != tt.expected {
				t.Errorf("Expected %d workers, got %d", tt.expected, pool.workers)
			}
			if pool.serverAddr != tt.serverAddr {
				t.Errorf("Expected serverAddr %s, got %s", tt.serverAddr, pool.serverAddr)
			}
			if pool.hashKey != tt.hashKey {
				t.Errorf("Expected hashKey %s, got %s", tt.hashKey, pool.hashKey)
			}
			if pool.client == nil {
				t.Error("Expected client to be initialized")
			}
			if cap(pool.jobs) != tt.expected*100 {
				t.Errorf("Expected job queue capacity %d, got %d", tt.expected*100, cap(pool.jobs))
			}
		})
	}
}

func TestWorkerPoolDoRequest(t *testing.T) {
	t.Run("successful request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		pool := NewWorkerPool(1, server.Listener.Addr().String(), "")
		body := []byte(`{"test":"data"}`)

		err := pool.doRequest(server.URL+"/update", body, false)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("request with hash", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("HashSHA256") == "" {
				t.Error("Expected HashSHA256 header")
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		pool := NewWorkerPool(1, server.Listener.Addr().String(), "test-key")
		body := []byte(`{"data":"test"}`)

		err := pool.doRequest(server.URL+"/update", body, true)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("request with hash but empty key", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("HashSHA256") != "" {
				t.Error("Expected no HashSHA256 header when key is empty")
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		pool := NewWorkerPool(1, server.Listener.Addr().String(), "")
		body := []byte(`{"data":"test"}`)

		err := pool.doRequest(server.URL+"/update", body, true)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("server returns error status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("bad request"))
		}))
		defer server.Close()

		pool := NewWorkerPool(1, server.Listener.Addr().String(), "")
		body := []byte(`{"test":"data"}`)

		err := pool.doRequest(server.URL+"/update", body, false)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !strings.Contains(err.Error(), "unexpected status code") {
			t.Errorf("Expected error about status code, got %v", err)
		}
	})

	t.Run("invalid URL", func(t *testing.T) {
		pool := NewWorkerPool(1, "invalid-url", "")
		body := []byte(`{"test":"data"}`)

		err := pool.doRequest("http://invalid-url-that-does-not-exist:9999/update", body, false)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestCompressData(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "normal data",
			data:    []byte("hello world"),
			wantErr: false,
		},
		{
			name:    "empty data",
			data:    []byte(""),
			wantErr: false,
		},
		{
			name:    "large data",
			data:    bytes.Repeat([]byte("a"), 100000),
			wantErr: false,
		},
		{
			name:    "binary data",
			data:    []byte{0x00, 0x01, 0x02, 0xFF},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressed, err := compressData(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("compressData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				gzipReader, err := gzip.NewReader(bytes.NewReader(compressed))
				if err != nil {
					t.Errorf("Failed to create gzip reader: %v", err)
					return
				}
				defer gzipReader.Close()

				decompressed, err := io.ReadAll(gzipReader)
				if err != nil {
					t.Errorf("Failed to decompress: %v", err)
					return
				}

				if !bytes.Equal(decompressed, tt.data) {
					t.Errorf("Decompressed data doesn't match original. Got %v, want %v", decompressed, tt.data)
				}
			}
		})
	}
}

func TestWorkerPoolClientConfiguration(t *testing.T) {
	pool := NewWorkerPool(1, "localhost:8080", "")

	if pool.client.Timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", pool.client.Timeout)
	}

	transport, ok := pool.client.Transport.(*http.Transport)
	if !ok {
		t.Error("Expected http.Transport")
	} else {
		if transport.MaxIdleConns != 100 {
			t.Errorf("Expected MaxIdleConns 100, got %d", transport.MaxIdleConns)
		}
		if transport.IdleConnTimeout != 90*time.Second {
			t.Errorf("Expected IdleConnTimeout 90s, got %v", transport.IdleConnTimeout)
		}
	}
}

func BenchmarkCompressData(b *testing.B) {
	data := bytes.Repeat([]byte("test data for compression"), 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		compressData(data)
	}
}
