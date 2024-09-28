package storage

import (
	"sync"

	"github.com/rafaeljusto/anicetus"
)

var _ anicetus.GatekeeperStorage = &InMemory{}

// InMemory is an in-memory storage for the fingerprints.
type InMemory struct {
	// data is the data stored in the storage.
	data sync.Map
}

// NewInMemory creates a new in-memory storage.
func NewInMemory() *InMemory {
	return &InMemory{}
}

// Exists checks if the fingerprint exists in the storage.
func (s *InMemory) Exists(fingerprint anicetus.Fingerprint) (bool, error) {
	_, ok := s.data.Load(fingerprint)
	return ok, nil
}

// Processed checks if the fingerprint was processed.
func (s *InMemory) Processed(fingerprint anicetus.Fingerprint) (bool, error) {
	data, ok := s.data.Load(fingerprint)
	if !ok {
		return false, nil
	}
	return data.(bool), nil
}

// Store stores the fingerprint in the storage.
func (s *InMemory) Store(fingerprint anicetus.Fingerprint, processed bool) error {
	s.data.Store(fingerprint, processed)
	return nil
}

// Remove removes the fingerprint from the storage.
func (s *InMemory) Remove(fingerprint anicetus.Fingerprint) error {
	s.data.Delete(fingerprint)
	return nil
}
