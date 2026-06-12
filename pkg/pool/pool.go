// Package pool предоставляет generic-пул для повторного использования объектов.
//
// Пул позволяет переиспользовать "тяжёлые" объекты, снижая нагрузку на сборщик мусора.
// Перед возвратом в пул объект сбрасывается через метод Reset().
//
// Пример использования:
//
//	type MyStruct struct {
//	    Data []string
//	}
//
//	func (m *MyStruct) Reset() {
//	    m.Data = m.Data[:0]
//	}
//
//	p := pool.New(func() *MyStruct { return &MyStruct{} })
//	obj := p.Get()
//	defer p.Put(obj)
//	obj.Data = append(obj.Data, "example")
package pool

import (
	"sync"
)

// Resetter - интерфейс для объектов, которые можно сбросить
type Resetter interface {
	Reset()
}

// Pool - generic-пул для объектов с методом Reset
type Pool[T Resetter] struct {
	p     sync.Pool
	newFn func() T
}

// New создает новый пул с функцией-конструктором
func New[T Resetter](newFn func() T) *Pool[T] {
	return &Pool[T]{
		newFn: newFn,
		p: sync.Pool{
			New: func() any {
				return newFn()
			},
		},
	}
}

// Get возвращает объект из пула.
// Если пул пуст, создается новый объект через функцию-конструктор.
func (p *Pool[T]) Get() T {
	return p.p.Get().(T)
}

// Put возвращает объект в пул после сброса его состояния.
func (p *Pool[T]) Put(item T) {
	item.Reset()
	p.p.Put(item)
}

// PutWithoutReset возвращает объект в пул без сброса.
// Используйте только если уверены, что сброс не нужен.
func (p *Pool[T]) PutWithoutReset(item T) {
	p.p.Put(item)
}
