package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponseWriterHelper_JSON(t *testing.T) {
	helper := &ResponseWriterHelper{}

	tests := []struct {
		name       string
		statusCode int
		data       interface{}
		checkFunc  func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:       "success with data",
			statusCode: http.StatusOK,
			data:       map[string]string{"message": "success"},
			checkFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Code != http.StatusOK {
					t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
				}
				if w.Header().Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
				}

				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Fatal(err)
				}
				if response["message"] != "success" {
					t.Errorf("Expected message 'success', got '%s'", response["message"])
				}
			},
		},
		{
			name:       "success with nil data",
			statusCode: http.StatusNoContent,
			data:       nil,
			checkFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Code != http.StatusNoContent {
					t.Errorf("Expected status %d, got %d", http.StatusNoContent, w.Code)
				}
				if w.Header().Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
				}
				if w.Body.Len() != 0 {
					t.Errorf("Expected empty body for nil data, got %s", w.Body.String())
				}
			},
		},
		{
			name:       "created status",
			statusCode: http.StatusCreated,
			data:       map[string]int{"id": 123},
			checkFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Code != http.StatusCreated {
					t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
				}

				var response map[string]int
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Fatal(err)
				}
				if response["id"] != 123 {
					t.Errorf("Expected id 123, got %d", response["id"])
				}
			},
		},
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			data:       map[string]string{"error": "invalid input"},
			checkFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Code != http.StatusBadRequest {
					t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
				}

				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Fatal(err)
				}
				if response["error"] != "invalid input" {
					t.Errorf("Expected error 'invalid input', got '%s'", response["error"])
				}
			},
		},
		{
			name:       "complex data structure",
			statusCode: http.StatusOK,
			data: struct {
				Name  string `json:"name"`
				Value int    `json:"value"`
				Tags  []string `json:"tags"`
			}{
				Name:  "test",
				Value: 42,
				Tags:  []string{"tag1", "tag2"},
			},
			checkFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response struct {
					Name  string   `json:"name"`
					Value int      `json:"value"`
					Tags  []string `json:"tags"`
				}
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Fatal(err)
				}
				if response.Name != "test" {
					t.Errorf("Expected name 'test', got '%s'", response.Name)
				}
				if response.Value != 42 {
					t.Errorf("Expected value 42, got %d", response.Value)
				}
				if len(response.Tags) != 2 || response.Tags[0] != "tag1" || response.Tags[1] != "tag2" {
					t.Errorf("Expected tags [tag1 tag2], got %v", response.Tags)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			helper.JSON(w, tt.statusCode, tt.data)
			tt.checkFunc(t, w)
		})
	}
}

func TestResponseWriterHelper_Error(t *testing.T) {
	helper := &ResponseWriterHelper{}

	tests := []struct {
		name       string
		statusCode int
		message    string
	}{
		{
			name:       "bad request error",
			statusCode: http.StatusBadRequest,
			message:    "invalid request",
		},
		{
			name:       "not found error",
			statusCode: http.StatusNotFound,
			message:    "resource not found",
		},
		{
			name:       "internal server error",
			statusCode: http.StatusInternalServerError,
			message:    "something went wrong",
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			message:    "authentication required",
		},
		{
			name:       "forbidden",
			statusCode: http.StatusForbidden,
			message:    "access denied",
		},
		{
			name:       "conflict",
			statusCode: http.StatusConflict,
			message:    "resource already exists",
		},
		{
			name:       "too many requests",
			statusCode: http.StatusTooManyRequests,
			message:    "rate limit exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			helper.Error(w, tt.statusCode, tt.message)

			if w.Code != tt.statusCode {
				t.Errorf("Expected status %d, got %d", tt.statusCode, w.Code)
			}

			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
			}

			var response map[string]string
			err := json.NewDecoder(w.Body).Decode(&response)
			if err != nil {
				t.Fatal(err)
			}

			if response["error"] != tt.message {
				t.Errorf("Expected error message '%s', got '%s'", tt.message, response["error"])
			}
		})
	}
}

func TestResponseWriterHelper_JSON_EncodingError(t *testing.T) {
	helper := &ResponseWriterHelper{}
	w := httptest.NewRecorder()

	data := make(chan int)

	helper.JSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	t.Logf("Response body on encoding error: %q", w.Body.String())
}

func TestResponseWriterHelper_JSON_EdgeCases(t *testing.T) {
	helper := &ResponseWriterHelper{}

	t.Run("nil writer should panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic with nil writer")
			}
		}()
		helper.JSON(nil, http.StatusOK, map[string]string{"test": "data"})
	})

	t.Run("zero status code", func(t *testing.T) {
		w := httptest.NewRecorder()
		helper.JSON(w, 0, map[string]string{"test": "data"})
		if w.Code != http.StatusOK {
			t.Errorf("Expected default 200 OK for status 0, got %d", w.Code)
		}
	})
}

func BenchmarkResponseWriterHelper_JSON(b *testing.B) {
	helper := &ResponseWriterHelper{}
	data := map[string]string{"key": "value"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		helper.JSON(w, http.StatusOK, data)
	}
}

func BenchmarkResponseWriterHelper_Error(b *testing.B) {
	helper := &ResponseWriterHelper{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		helper.Error(w, http.StatusBadRequest, "error message")
	}
}

func ExampleResponseWriterHelper_JSON() {
	helper := &ResponseWriterHelper{}
	w := httptest.NewRecorder()

	data := map[string]string{"status": "ok"}
	helper.JSON(w, http.StatusOK, data)

}

func ExampleResponseWriterHelper_Error() {
	helper := &ResponseWriterHelper{}
	w := httptest.NewRecorder()

	helper.Error(w, http.StatusNotFound, "resource not found")
