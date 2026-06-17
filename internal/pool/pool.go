// Package pool предоставляет generic-обёртку над sync.Pool для объектов с методом Reset.
package pool

import "sync"

// Pool хранит объекты одного типа, которые можно сбрасывать и повторно использовать.
type Pool[T interface{ Reset() }] struct {
	pool sync.Pool
}

// New создаёт и возвращает указатель на новый пул объектов.
func New[T any, PT interface {
	*T
	Reset()
}]() *Pool[PT] {
	return &Pool[PT]{
		pool: sync.Pool{
			New: func() any {
				return PT(new(T))
			},
		},
	}
}

// Get возвращает объект из пула.
func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put помещает объект в пул, предварительно сбросив его состояние.
func (p *Pool[T]) Put(x T) {
	x.Reset()
	p.pool.Put(x)
}
