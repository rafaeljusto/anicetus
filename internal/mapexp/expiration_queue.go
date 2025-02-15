package mapexp

import (
	"sync"
	"time"
)

type expirationQueueItem[K comparable] struct {
	key        K
	expiration time.Time
}

type expirationQueue[K comparable] struct {
	items []expirationQueueItem[K]
	ttl   time.Duration
	mutex sync.Mutex
}

func newExpirationQueue[K comparable](ttl time.Duration) *expirationQueue[K] {
	return &expirationQueue[K]{
		ttl: ttl,
	}
}

func (e *expirationQueue[K]) add(key K) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	for i, item := range e.items {
		if item.key == key {
			e.items = append(e.items[:i], e.items[i+1:]...)
			break
		}
	}

	e.items = append(e.items, expirationQueueItem[K]{
		key:        key,
		expiration: time.Now().Add(e.ttl),
	})
}

func (e *expirationQueue[K]) renew(key K) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	for i, item := range e.items {
		if item.key == key {
			e.items[i].expiration = time.Now().Add(e.ttl)
			break
		}
	}
}

func (e *expirationQueue[K]) purge() []K {
	var expiredKeys []K

	e.mutex.Lock()
	defer e.mutex.Unlock()

	now := time.Now()

	var i int
	for i = 0; i < len(e.items); i++ {
		if e.items[i].expiration.After(now) {
			break
		}
		expiredKeys = append(expiredKeys, e.items[i].key)
	}
	e.items = e.items[i:]

	return expiredKeys
}
