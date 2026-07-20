package storage

import (
	"context"
	"sync"
	"time"

	"github.com/rafaeljusto/anicetus/v2"
)

var _ anicetus.GatekeeperStorage = &InMemory{}

// inMemorySweepInterval is the number of mutating operations between opportunistic
// sweeps of expired entries. It bounds memory without a background goroutine.
const inMemorySweepInterval = 1024

// inMemoryEntry is a stored fingerprint together with its lease expiration.
type inMemoryEntry struct {
	processed bool
	expiresAt time.Time
}

// InMemory is an in-memory storage for the fingerprints.
type InMemory struct {
	// mu guards data and opsSinceSweep, making the check-and-set in Add atomic.
	mu   sync.Mutex
	data map[anicetus.Fingerprint]inMemoryEntry
	// leaseTTL is how long an entry is held before it is considered expired.
	leaseTTL time.Duration
	// opsSinceSweep counts mutating operations since the last expiry sweep.
	opsSinceSweep int
}

// NewInMemory creates a new in-memory storage.
func NewInMemory(options ...Option) *InMemory {
	o := NewOptions()
	for _, opt := range options {
		opt(o)
	}

	return &InMemory{
		data:     make(map[anicetus.Fingerprint]inMemoryEntry),
		leaseTTL: o.LeaseTTL(),
	}
}

// Add atomically records the fingerprint as in-flight if it is not already
// present (and unexpired), returning true only for the caller that created the
// entry. This is the election primitive: under a thundering herd exactly one
// concurrent caller receives true.
func (s *InMemory) Add(_ context.Context, fingerprint anicetus.Fingerprint) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if e, ok := s.data[fingerprint]; ok && now.Before(e.expiresAt) {
		return false, nil
	}

	s.data[fingerprint] = inMemoryEntry{
		processed: false,
		expiresAt: now.Add(s.leaseTTL),
	}
	s.sweep(now)
	return true, nil
}

// Processed checks if the fingerprint was processed.
func (s *InMemory) Processed(_ context.Context, fingerprint anicetus.Fingerprint) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.data[fingerprint]
	if !ok || !time.Now().Before(e.expiresAt) {
		return false, nil
	}
	return e.processed, nil
}

// Store stores the fingerprint in the storage, refreshing its lease.
func (s *InMemory) Store(_ context.Context, fingerprint anicetus.Fingerprint, processed bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.data[fingerprint] = inMemoryEntry{
		processed: processed,
		expiresAt: now.Add(s.leaseTTL),
	}
	s.sweep(now)
	return nil
}

// Remove removes the fingerprint from the storage.
func (s *InMemory) Remove(_ context.Context, fingerprint anicetus.Fingerprint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, fingerprint)
	return nil
}

// sweep opportunistically deletes expired entries. The caller MUST hold s.mu.
// It runs at most once every inMemorySweepInterval mutating operations, so the
// amortized cost is negligible while keeping the map bounded to live entries.
func (s *InMemory) sweep(now time.Time) {
	s.opsSinceSweep++
	if s.opsSinceSweep < inMemorySweepInterval {
		return
	}
	s.opsSinceSweep = 0

	for fingerprint, e := range s.data {
		if !now.Before(e.expiresAt) {
			delete(s.data, fingerprint)
		}
	}
}
