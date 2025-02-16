package mapexp

import (
	"sync"
	"time"
)

// Map is a generic map with items that expire after a certain duration.
type Map[K comparable, V any] struct {
	items           map[K]V
	itemsMutex      sync.RWMutex
	expirationQueue *expirationQueue[K]

	stop chan struct{}
}

// New creates a new Map.
func New[K comparable, V any](ttl time.Duration) *Map[K, V] {
	m := &Map[K, V]{
		items:           make(map[K]V),
		expirationQueue: newExpirationQueue[K](ttl),
	}
	m.start()
	return m
}

// Set sets the value for the key in the map.
func (m *Map[K, V]) Set(key K, value V) {
	m.itemsMutex.Lock()
	m.items[key] = value
	m.itemsMutex.Unlock()

	m.expirationQueue.add(key)
}

// Get gets the value for the key in the map.
func (m *Map[K, V]) Get(key K) (V, bool) {
	m.itemsMutex.RLock()
	value, ok := m.items[key]
	m.itemsMutex.RUnlock()

	m.expirationQueue.renew(key)
	return value, ok
}

// Delete deletes the key from the map.
func (m *Map[K, V]) Delete(key K) {
	m.itemsMutex.Lock()
	delete(m.items, key)
	m.itemsMutex.Unlock()
}

func (m *Map[K, V]) start() {
	go func() {
		timer := time.NewTicker(m.expirationQueue.ttl)
		defer timer.Stop()

		for {
			select {
			case <-m.stop:
				return
			case <-timer.C:
			}

			expiredKeys := m.expirationQueue.purge()

			m.itemsMutex.Lock()
			for _, key := range expiredKeys {
				delete(m.items, key)
			}
			m.itemsMutex.Unlock()
		}
	}()
}

// Stop stops the map from expiring the keys.
func (m *Map[K, V]) Stop() {
	close(m.stop)
}
