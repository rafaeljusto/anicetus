package detector

import (
	"context"

	"github.com/rafaeljusto/anicetus/v2"
	"github.com/rafaeljusto/anicetus/v2/internal/mapexp"
)

var _ anicetus.Detector = &SingleFlightInMemory{}

// SingleFlightInMemory is a detector strategy that coalesces every concurrent
// duplicate of a fingerprint, storing the state in memory.
//
// Unlike the token bucket strategy, it has no burst threshold to cross: it
// treats a fingerprint as a thundering herd from the very first duplicate. The
// gatekeeper then elects a single request to reach the backend (StatusProcess)
// and makes every concurrent request with the same fingerprint wait
// (StatusWait) until that request finishes. This is the deterministic
// equivalent of golang.org/x/sync/singleflight, but coordinated through the
// gatekeeper storage so it can work across processes (see the Redis storage).
//
// Use this when you want to guarantee that no more than one request per
// fingerprint reaches the backend at a time, even for a herd of two. Prefer the
// token bucket strategy when a burst of concurrent requests is acceptable and
// you only want to engage once traffic crosses a threshold.
//
// Every distinct fingerprint touches the gatekeeper storage once per cooldown
// window, so memory/storage usage grows with fingerprint cardinality. Keep the
// fingerprint aligned with the backend cache key to avoid over-coalescing
// unrelated requests.
type SingleFlightInMemory struct {
	cooldowns *mapexp.Map[anicetus.Fingerprint, bool]
}

// NewSingleFlightInMemory creates a new single-flight detector strategy.
func NewSingleFlightInMemory(options ...SingleFlightOption) *SingleFlightInMemory {
	o := NewSingleFlightOptions()
	for _, opt := range options {
		opt(o)
	}

	return &SingleFlightInMemory{
		cooldowns: mapexp.New[anicetus.Fingerprint, bool](o.CoolDownInterval()),
	}
}

// CoolDown will cool down the fingerprint.
func (s *SingleFlightInMemory) CoolDown(_ context.Context, fingerprint anicetus.Fingerprint) error {
	s.cooldowns.Set(fingerprint, true)
	return nil
}

// IsCoolDown checks if the fingerprint is in cooldown.
func (s *SingleFlightInMemory) IsCoolDown(_ context.Context, fingerprint anicetus.Fingerprint) (bool, error) {
	cooldown, ok := s.cooldowns.Get(fingerprint)
	return cooldown && ok, nil
}

// IsThunderingHerd always reports a thundering herd. Coalescing is delegated to
// the gatekeeper, which lets the first request through and blocks the rest, so
// there is no threshold to detect here. Requests are only allowed straight
// through while the fingerprint is in cooldown (see Evaluate), which is when the
// backend cache is assumed to be warm.
func (s *SingleFlightInMemory) IsThunderingHerd(_ context.Context, _ anicetus.Fingerprint) (bool, error) {
	return true, nil
}
