package anicetus

import "fmt"

// Gatekeeper stores the logic to control the thundering herd problem.
type Gatekeeper struct {
	// storage is the storage to keep track of the fingerprints.
	storage GatekeeperStorage
}

// NewGatekeeper creates a new gatekeeper.
func NewGatekeeper(storage GatekeeperStorage) *Gatekeeper {
	return &Gatekeeper{
		storage: storage,
	}
}

// analyze checks if the fingerprint is valid to be processed.
func (g Gatekeeper) analyze(fingerprint Fingerprint) (Status, error) {
	if exists, err := g.storage.Exists(fingerprint); err != nil {
		return StatusFailed, fmt.Errorf("failed to check if fingerprint exists: %w", err)

	} else if exists {
		if processed, err := g.storage.Processed(fingerprint); err != nil {
			return StatusFailed, fmt.Errorf("failed to get fingerprint processed flag: %w", err)

		} else if processed {
			return StatusOpenGates, nil
		}

		return StatusWait, nil
	}

	if err := g.storage.Store(fingerprint, false); err != nil {
		return StatusFailed, fmt.Errorf("failed to store fingerprint: %w", err)
	}

	return StatusProcess, nil
}

// Store stores the fingerprint in the storage. This should be called after the
// processing is done of the StatusProcess.
func (g Gatekeeper) Store(fingerprint Fingerprint, processed bool) error {
	return g.storage.Store(fingerprint, processed)
}

// Remove removes the fingerprint from the storage. This should be called in
// case there is some error while processing the request.
func (g Gatekeeper) Remove(fingerprint Fingerprint) error {
	return g.storage.Remove(fingerprint)
}

// GatekeeperStorage stores the fingerprints.
type GatekeeperStorage interface {
	// Exists checks if the fingerprint exists in the storage.
	Exists(fingerprint Fingerprint) (bool, error)
	// Processed checks if the fingerprint has been processed.
	Processed(fingerprint Fingerprint) (bool, error)
	// Store stores the fingerprint in the storage.
	Store(fingerprint Fingerprint, processed bool) error
	// Remove removes the fingerprint from the storage.
	Remove(fingerprint Fingerprint) error
}
