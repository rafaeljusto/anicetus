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
	// Add is the atomic election: exactly one concurrent request with the same
	// fingerprint gets elected == true and is allowed to reach the backend.
	elected, err := g.storage.Add(ctx, fingerprint)
	if err != nil {
		return StatusFailed, fmt.Errorf("failed to elect fingerprint: %w", err)
	}
	if elected {
		return StatusProcess, nil
	}

	// Another request was already elected. Let this one through only once the
	// elected request has populated the backend cache, otherwise make it wait.
	processed, err := g.storage.Processed(ctx, fingerprint)
	if err != nil {
		return StatusFailed, fmt.Errorf("failed to get fingerprint processed flag: %w", err)
	}
	if processed {
		return StatusOpenGates, nil
	}

	return StatusWait, nil
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
	// Add atomically records the fingerprint as in-flight (stored with the
	// processed flag set to false) if it is not already present, returning true
	// only when this call created the entry. For a group of concurrent requests
	// with the same fingerprint, exactly one caller MUST receive true: that
	// request is the one elected to reach the backend. This method is the
	// election primitive and therefore MUST be atomic (check-and-set), otherwise
	// a thundering herd can elect more than one request.
	//
	// The entry created by Add MUST expire after an implementation-defined lease.
	// This guarantees that if the elected request dies before calling Store or
	// Remove, the fingerprint is eventually released for a new election instead
	// of blocking every subsequent request forever.
	Add(ctx context.Context, fingerprint Fingerprint) (bool, error)
	// Processed checks if the fingerprint has been processed.
	Processed(ctx context.Context, fingerprint Fingerprint) (bool, error)
	// Store stores the fingerprint in the storage with the given processed flag.
	Store(ctx context.Context, fingerprint Fingerprint, processed bool) error
	// Remove removes the fingerprint from the storage. It MUST not return an
	// error if the fingerprint doesn't exist.
	Remove(ctx context.Context, fingerprint Fingerprint) error
}
