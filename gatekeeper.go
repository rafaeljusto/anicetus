package anicetus

import (
	"context"
	"fmt"
)

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
func (g Gatekeeper) analyze(ctx context.Context, fingerprint Fingerprint) (Status, error) {
	if exists, err := g.storage.Exists(ctx, fingerprint); err != nil {
		return StatusFailed, fmt.Errorf("failed to check if fingerprint exists: %w", err)

	} else if exists {
		if processed, err := g.storage.Processed(ctx, fingerprint); err != nil {
			return StatusFailed, fmt.Errorf("failed to get fingerprint processed flag: %w", err)

		} else if processed {
			return StatusOpenGates, nil
		}

		return StatusWait, nil
	}

	if err := g.storage.Store(ctx, fingerprint, false); err != nil {
		return StatusFailed, fmt.Errorf("failed to store fingerprint: %w", err)
	}

	return StatusProcess, nil
}

// Store stores the fingerprint in the storage. This should be called after the
// processing is done of the StatusProcess.
func (g Gatekeeper) Store(ctx context.Context, fingerprint Fingerprint, processed bool) error {
	return g.storage.Store(ctx, fingerprint, processed)
}

// Remove removes the fingerprint from the storage. This should be called in
// case there is some error while processing the request.
func (g Gatekeeper) Remove(ctx context.Context, fingerprint Fingerprint) error {
	return g.storage.Remove(ctx, fingerprint)
}

// GatekeeperStorage stores the fingerprints.
type GatekeeperStorage interface {
	// Exists checks if the fingerprint exists in the storage.
	Exists(ctx context.Context, fingerprint Fingerprint) (bool, error)
	// Processed checks if the fingerprint has been processed.
	Processed(ctx context.Context, fingerprint Fingerprint) (bool, error)
	// Store stores the fingerprint in the storage.
	Store(ctx context.Context, fingerprint Fingerprint, processed bool) error
	// Remove removes the fingerprint from the storage. It MUST not return an
	// error if the fingerprint doesn't exist.
	Remove(ctx context.Context, fingerprint Fingerprint) error
}
