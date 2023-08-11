package sync_generics

import "sync"

type Map[K any, V any] struct {
	inner sync.Map
}

func (m *Map[K, V]) CompareAndDelete(key K, old V) (deleted bool) {
	deleted = m.inner.CompareAndDelete(key, old)
	return
}

func (m *Map[K, V]) CompareAndSwap(key K, old V, new V) bool {
	return m.inner.CompareAndSwap(key, old, new)
}

func (m *Map[K, V]) Delete(key K) {
	m.inner.Delete(key)
}

func (m *Map[K, V]) Load(key K) (value V, ok bool) {
	_value, ok := m.inner.Load(key)
	value = nilSafeTypeAssertion[V](_value)
	return
}

func (m *Map[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	_value, loaded := m.inner.LoadAndDelete(key)
	value = nilSafeTypeAssertion[V](_value)
	return
}

func (m *Map[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	_actual, loaded := m.inner.LoadOrStore(key, value)
	actual = nilSafeTypeAssertion[V](_actual)
	return
}

func (m *Map[K, V]) Range(f func(key K, value V) bool) {
	m.inner.Range(func(key, value any) bool {
		return f(nilSafeTypeAssertion[K](key), nilSafeTypeAssertion[V](value))
	})
}

func (m *Map[K, V]) Store(key K, value V) {
	m.inner.Store(key, value)
}

func (m *Map[K, V]) Swap(key K, value V) (previous V, loaded bool) {
	_previous, loaded := m.inner.Swap(key, value)
	previous = nilSafeTypeAssertion[V](_previous)
	return
}

func nilSafeTypeAssertion[T any](value any) T {
	var zero T
	if value == nil {
		return zero
	}
	return value.(T)
}
