package ebp

import "sync"

type RwMap[K comparable, V any] struct {
	store map[K]V
	mu    sync.RWMutex
}

func NewRwMap[K comparable, V any]() RwMap[K, V] {
	return RwMap[K, V]{
		store: make(map[K]V),
		mu:    sync.RWMutex{},
	}
}

func (m *RwMap[K, V]) Set(k K, v V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[k] = v
}

func (m *RwMap[K, V]) Get(k K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, exist := m.store[k]
	return v, exist
}

func (m *RwMap[K, V]) Keys() []K {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]K, 0, len(m.store))
	for k := range m.store {
		res = append(res, k)
	}
	return res
}
