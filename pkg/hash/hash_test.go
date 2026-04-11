package hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestComputeHMAC(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		key      string
		expected string
	}{
		{
			name:     "normal case with valid key",
			data:     []byte("hello world"),
			key:      "secret-key",
			expected: computeExpectedHMAC("hello world", "secret-key"),
		},
		{
			name:     "empty data",
			data:     []byte(""),
			key:      "secret-key",
			expected: computeExpectedHMAC("", "secret-key"),
		},
		{
			name:     "empty key returns empty string",
			data:     []byte("hello world"),
			key:      "",
			expected: "",
		},
		{
			name:     "long key",
			data:     []byte("test data"),
			key:      "very-long-key-that-exceeds-block-size-of-sha256-which-is-64-bytes-let-me-add-more-text-to-make-it-really-long",
			expected: computeExpectedHMAC("test data", "very-long-key-that-exceeds-block-size-of-sha256-which-is-64-bytes-let-me-add-more-text-to-make-it-really-long"),
		},
		{
			name:     "special characters in data",
			data:     []byte("!@#$%^&*()_+{}|:<>?~`"),
			key:      "key123",
			expected: computeExpectedHMAC("!@#$%^&*()_+{}|:<>?~`", "key123"),
		},
		{
			name:     "binary data",
			data:     []byte{0x00, 0x01, 0x02, 0xFF, 0xFE},
			key:      "binary-key",
			expected: computeExpectedHMAC(string([]byte{0x00, 0x01, 0x02, 0xFF, 0xFE}), "binary-key"),
		},
		{
			name:     "unicode data",
			data:     []byte("Привет мир"),
			key:      "unicode-key",
			expected: computeExpectedHMAC("Привет мир", "unicode-key"),
		},
		{
			name:     "numeric key",
			data:     []byte("data"),
			key:      "1234567890",
			expected: computeExpectedHMAC("data", "1234567890"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeHMAC(tt.data, tt.key)
			if result != tt.expected {
				t.Errorf("ComputeHMAC() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestVerifyHMAC(t *testing.T) {
	data := []byte("test data")
	key := "secret-key"
	validHash := ComputeHMAC(data, key)

	tests := []struct {
		name     string
		data     []byte
		hash     string
		key      string
		expected bool
	}{
		{
			name:     "valid HMAC",
			data:     data,
			hash:     validHash,
			key:      key,
			expected: true,
		},
		{
			name:     "invalid HMAC",
			data:     data,
			hash:     "invalidhash123",
			key:      key,
			expected: false,
		},
		{
			name:     "empty key returns true",
			data:     data,
			hash:     "anyhash",
			key:      "",
			expected: true,
		},
		{
			name:     "empty key with valid hash returns true",
			data:     data,
			hash:     validHash,
			key:      "",
			expected: true,
		},
		{
			name:     "wrong data",
			data:     []byte("wrong data"),
			hash:     validHash,
			key:      key,
			expected: false,
		},
		{
			name:     "wrong key",
			data:     data,
			hash:     validHash,
			key:      "wrong-key",
			expected: false,
		},
		{
			name:     "empty hash",
			data:     data,
			hash:     "",
			key:      key,
			expected: false,
		},
		{
			name:     "nil data",
			data:     nil,
			hash:     ComputeHMAC(nil, key),
			key:      key,
			expected: true,
		},
		{
			name:     "case sensitivity test",
			data:     data,
			hash:     strings.ToUpper(validHash),
			key:      key,
			expected: false,
		},
		{
			name:     "hash with extra characters",
			data:     data,
			hash:     validHash + "extra",
			key:      key,
			expected: false,
		},
		{
			name:     "hash with missing characters",
			data:     data,
			hash:     validHash[:len(validHash)-5],
			key:      key,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VerifyHMAC(tt.data, tt.hash, tt.key)
			if result != tt.expected {
				t.Errorf("VerifyHMAC() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestComputeAndVerifyIntegration(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
		key  string
	}{
		{
			name: "simple string",
			data: []byte("hello world"),
			key:  "secret",
		},
		{
			name: "empty data",
			data: []byte(""),
			key:  "secret",
		},
		{
			name: "large data",
			data: []byte(strings.Repeat("a", 10000)),
			key:  "secret",
		},
		{
			name: "special characters",
			data: []byte("data with spaces and punctuation!"),
			key:  "key-with-dashes",
		},
		{
			name: "json data",
			data: []byte(`{"user":"alice","action":"login"}`),
			key:  "api-key-123",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hash := ComputeHMAC(tc.data, tc.key)

			if !VerifyHMAC(tc.data, hash, tc.key) {
				t.Errorf("Verification failed for correct hash")
			}

			if VerifyHMAC(tc.data, hash, "wrong-key") {
				t.Errorf("Verification succeeded with wrong key")
			}

			if VerifyHMAC([]byte("wrong data"), hash, tc.key) {
				t.Errorf("Verification succeeded with wrong data")
			}

			if len(hash) > 0 {
				tamperedHash := hash[:len(hash)-1] + "0"
				if VerifyHMAC(tc.data, tamperedHash, tc.key) {
					t.Errorf("Verification succeeded with tampered hash")
				}
			}
		})
	}
}

func TestComputeHMACConsistency(t *testing.T) {
	data := []byte("consistent data")
	key := "consistent-key"

	hash1 := ComputeHMAC(data, key)
	hash2 := ComputeHMAC(data, key)

	if hash1 != hash2 {
		t.Errorf("HMAC not consistent: got %v and %v", hash1, hash2)
	}

	hash3 := ComputeHMAC(data, "different-key")
	if hash1 == hash3 {
		t.Errorf("Different keys produced same HMAC")
	}

	hash4 := ComputeHMAC([]byte("different data"), key)
	if hash1 == hash4 {
		t.Errorf("Different data produced same HMAC")
	}
}

func TestEdgeCases(t *testing.T) {
	t.Run("very long data", func(t *testing.T) {
		data := []byte(strings.Repeat("x", 1000000))
		key := "test-key"
		hash := ComputeHMAC(data, key)

		if hash == "" {
			t.Errorf("ComputeHMAC returned empty for large data")
		}

		if !VerifyHMAC(data, hash, key) {
			t.Errorf("Verification failed for large data")
		}
	})

	t.Run("very long key", func(t *testing.T) {
		data := []byte("test data")
		key := strings.Repeat("k", 1000)
		hash := ComputeHMAC(data, key)

		if hash == "" {
			t.Errorf("ComputeHMAC returned empty for large key")
		}

		if !VerifyHMAC(data, hash, key) {
			t.Errorf("Verification failed for large key")
		}
	})

	t.Run("nil vs empty slice", func(t *testing.T) {
		key := "test"
		var nilData []byte
		emptyData := []byte{}

		nilHash := ComputeHMAC(nilData, key)
		emptyHash := ComputeHMAC(emptyData, key)

		if nilHash != emptyHash {
			t.Errorf("Nil and empty slice produced different hashes: %v vs %v", nilHash, emptyHash)
		}
	})

	t.Run("hash length validation", func(t *testing.T) {
		data := []byte("test")
		key := "key"
		hash := ComputeHMAC(data, key)

		expectedLen := 64
		if len(hash) != expectedLen {
			t.Errorf("Expected hash length %d, got %d", expectedLen, len(hash))
		}
	})
}

func TestTimingAttackResistance(t *testing.T) {

	data := []byte("sensitive data")
	key := "secret"
	correctHash := ComputeHMAC(data, key)

	wrongHash := strings.Repeat("0", len(correctHash))

	correctResult := VerifyHMAC(data, correctHash, key)
	wrongResult := VerifyHMAC(data, wrongHash, key)

	if !correctResult {
		t.Errorf("Correct hash verification failed")
	}
	if wrongResult {
		t.Errorf("Wrong hash verification succeeded")
	}
}

func TestConcurrentAccess(t *testing.T) {
	data := []byte("concurrent test")
	key := "concurrent-key"

	done := make(chan bool)

	for i := 0; i < 100; i++ {
		go func() {
			hash := ComputeHMAC(data, key)
			if !VerifyHMAC(data, hash, key) {
				t.Errorf("Concurrent verification failed")
			}
			done <- true
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}

func computeExpectedHMAC(data, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func BenchmarkComputeHMAC(b *testing.B) {
	data := []byte("benchmark data for HMAC computation")
	key := "benchmark-key"

	for i := 0; i < b.N; i++ {
		ComputeHMAC(data, key)
	}
}

func BenchmarkComputeHMACLargeData(b *testing.B) {
	data := []byte(strings.Repeat("x", 100000))
	key := "benchmark-key"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ComputeHMAC(data, key)
	}
}

func BenchmarkVerifyHMAC(b *testing.B) {
	data := []byte("benchmark data")
	key := "benchmark-key"
	hash := ComputeHMAC(data, key)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		VerifyHMAC(data, hash, key)
	}
}

func BenchmarkVerifyHMACInvalid(b *testing.B) {
	data := []byte("benchmark data")
	key := "benchmark-key"
	hash := ComputeHMAC(data, key)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		VerifyHMAC(data, hash+"0", key)
	}
}

func ExampleComputeHMAC() {
	data := []byte("message to authenticate")
	key := "secret-key"

	hash := ComputeHMAC(data, key)
	println("HMAC:", hash)
}

func ExampleVerifyHMAC() {
	data := []byte("message to authenticate")
	key := "secret-key"

	hash := ComputeHMAC(data, key)

	if VerifyHMAC(data, hash, key) {
		println("Message is authentic")
	} else {
		println("Message has been tampered with")
	}
}
