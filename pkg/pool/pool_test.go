package pool

import (
	"sync"
	"testing"
)

// TestStruct - тестовая структура с методом Reset
type TestStruct struct {
	ID      int
	Name    string
	Data    []string
	Counter *int
	Items   map[string]int
}

func (t *TestStruct) Reset() {
	t.ID = 0
	t.Name = ""
	t.Data = t.Data[:0]
	if t.Counter != nil {
		*t.Counter = 0
	}
	clear(t.Items)
}

func newTestStruct() *TestStruct {
	counter := 0
	return &TestStruct{
		Items:   make(map[string]int),
		Counter: &counter,
	}
}

func TestNew(t *testing.T) {
	pool := New(newTestStruct)

	if pool == nil {
		t.Fatal("Expected non-nil pool")
	}

	if pool.newFn == nil {
		t.Error("Expected non-nil newFn")
	}
}

func TestGet(t *testing.T) {
	pool := New(newTestStruct)

	obj := pool.Get()
	if obj == nil {
		t.Fatal("Expected non-nil object")
	}

	// Проверяем, что объект инициализирован
	if obj.Counter == nil {
		t.Error("Expected non-nil Counter")
	}
	if obj.Items == nil {
		t.Error("Expected non-nil Items")
	}
}

func TestPut(t *testing.T) {
	pool := New(newTestStruct)

	// Получаем объект и модифицируем его
	obj := pool.Get()
	obj.ID = 42
	obj.Name = "test"
	obj.Data = append(obj.Data, "hello", "world")
	*obj.Counter = 100
	obj.Items["key"] = 1

	// Возвращаем в пул (должен сбросить состояние)
	pool.Put(obj)

	// Получаем снова и проверяем, что состояние сброшено
	obj2 := pool.Get()
	if obj2.ID != 0 {
		t.Errorf("Expected ID 0, got %d", obj2.ID)
	}
	if obj2.Name != "" {
		t.Errorf("Expected Name '', got '%s'", obj2.Name)
	}
	if len(obj2.Data) != 0 {
		t.Errorf("Expected empty Data, got %v", obj2.Data)
	}
	if *obj2.Counter != 0 {
		t.Errorf("Expected Counter 0, got %d", *obj2.Counter)
	}
	if len(obj2.Items) != 0 {
		t.Errorf("Expected empty Items, got %v", obj2.Items)
	}
}

func TestPutWithoutReset(t *testing.T) {
	pool := New(newTestStruct)

	obj := pool.Get()
	obj.ID = 42
	obj.Name = "test"

	pool.PutWithoutReset(obj)

	obj2 := pool.Get()
	if obj2.ID != 42 {
		t.Errorf("Expected ID 42, got %d", obj2.ID)
	}
	if obj2.Name != "test" {
		t.Errorf("Expected Name 'test', got '%s'", obj2.Name)
	}
}

func TestConcurrent(t *testing.T) {
	pool := New(newTestStruct)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				obj := pool.Get()
				obj.ID = j
				obj.Data = append(obj.Data, "test")
				pool.Put(obj)
			}
		}()
	}

	wg.Wait()
}

func BenchmarkPool(b *testing.B) {
	pool := New(newTestStruct)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		obj := pool.Get()
		obj.ID = i
		obj.Data = append(obj.Data, "benchmark")
		pool.Put(obj)
	}
}

func BenchmarkWithoutPool(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		obj := newTestStruct()
		obj.ID = i
		obj.Data = append(obj.Data, "benchmark")
	}
}
