package anicetus

import (
	"fmt"
)

// Anicetus orchestrates the thundering herd detection and gatekeeping.
type Anicetus[F Fingerprinter] struct {
	// detector is the component that will be used to detect thundering herd.
	detector Detector
	// gatekeeper is the component that will be used to gatekeep thundering herd.
	gatekeeper *Gatekeeper
}

// NewAnicetus creates a new Anicetus.
func NewAnicetus[F Fingerprinter](detector Detector, gatekeeperStorage GatekeeperStorage) *Anicetus[F] {
	return &Anicetus[F]{
		detector:   detector,
		gatekeeper: NewGatekeeper(gatekeeperStorage),
	}
}

// Evaluate checks if the request is a thundering herd and if it is, it will
// gatekeep it.
func (t Anicetus[F]) Evaluate(f F) (Status, error) {
	fingerprint := f.Fingerprint()

	cooldown, err := t.detector.IsCoolDown(fingerprint)
	if err != nil {
		return StatusFailed, fmt.Errorf("failed to check if fingerprint is in cooldown: %w", err)
	} else if cooldown {
		return StatusOpenGates, nil
	}

	thunderingHerd, err := t.detector.IsThunderingHerd(fingerprint)
	if err != nil {
		return StatusFailed, fmt.Errorf("failed to check if fingerprint is a thundering herd: %w", err)
	} else if !thunderingHerd {
		return StatusOpenGates, nil
	}

	return t.gatekeeper.analyze(fingerprint)
}

// RequestDone will mark the request as done. This should be called after the
// request is processed.
func (t Anicetus[F]) RequestDone(f F) error {
	if err := t.gatekeeper.Store(f.Fingerprint(), true); err != nil {
		return fmt.Errorf("failed to store fingerprint: %w", err)
	}
	if err := t.detector.CoolDown(f.Fingerprint()); err != nil {
		return fmt.Errorf("failed to cooldown fingerprint: %w", err)
	}
	return nil
}

// Cleanup will remove the fingerprint from the storage. This should be called
// in case there is some error while processing the request.
func (t Anicetus[F]) Cleanup(f F) error {
	if err := t.gatekeeper.Remove(f.Fingerprint()); err != nil {
		return fmt.Errorf("failed to remove fingerprint: %w", err)
	}
	return nil
}

// Detector is the component that will be used to detect thundering herd.
type Detector interface {
	CoolDown(Fingerprint) error
	IsCoolDown(Fingerprint) (bool, error)
	IsThunderingHerd(Fingerprint) (bool, error)
}

// Fingerprint is the unique identifier for the request.
type Fingerprint string

// String returns the string representation of the fingerprint.
func (f Fingerprint) String() string {
	return string(f)
}

// Fingerprinter is the interface that wraps the Fingerprint method.
type Fingerprinter interface {
	Fingerprint() Fingerprint
}

// Status represents the status of the fingerprint.
type Status int

// List of possible statuses for the fingerprint.
const (
	StatusNone Status = iota

	// StatusFailed is returned when there is an error while processing the
	// request.
	StatusFailed

	// StatusProcess means that a thundering herd was detected and this request
	// was chosen to be processed by the backend. Once the request is done the
	// RequestDone method should be called.
	StatusProcess

	// StatusWait means that a thundering herd was detected and this request
	// should wait or be blocked until the initial request is done.
	StatusWait

	// StatusOpenGates is business as usual, meaning that no thundering herd is
	// happening or that the cooldown period is ongoing.
	StatusOpenGates
)

// String returns the string representation of the status.
func (s Status) String() string {
	switch s {
	case StatusNone:
		return "none"
	case StatusFailed:
		return "failed"
	case StatusProcess:
		return "process"
	case StatusWait:
		return "wait"
	case StatusOpenGates:
		return "open-gates"
	default:
		return "unknown"
	}
}
