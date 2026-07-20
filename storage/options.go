package storage

import (
	"log/slog"
	"time"
)

// defaultLeaseTTL is the default duration an elected (in-flight) fingerprint is
// held before it is considered stale and released for a new election. It should
// comfortably exceed the backend processing time so a slow-but-alive request is
// not evicted, while being short enough to recover quickly from a crashed one.
const defaultLeaseTTL = 30 * time.Second

// Options provides all the available options.
type Options struct {
	// logger to be used internally.
	logger *slog.Logger
	// leaseTTL is how long an elected fingerprint is held before expiring.
	leaseTTL time.Duration
}

// NewOptions creates a new Options with default values.
func NewOptions() *Options {
	return &Options{
		leaseTTL: defaultLeaseTTL,
	}
}

// Logger returns the logger to be used internally.
func (o *Options) Logger() *slog.Logger {
	return o.logger
}

// LeaseTTL returns the lease duration for an elected fingerprint.
func (o *Options) LeaseTTL() time.Duration {
	return o.leaseTTL
}

// Option is a helper function to configure the storage.
type Option func(*Options)

// WithLogger sets the logger to be used internally.
func WithLogger(logger *slog.Logger) Option {
	return func(o *Options) {
		o.logger = logger
	}
}

// WithLeaseTTL sets how long an elected fingerprint is held before it expires
// and becomes available for a new election. Set it above the expected backend
// processing time to avoid electing a second request while the first is still
// running, and below the point where a crashed request would block the
// fingerprint for an unacceptable amount of time.
func WithLeaseTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.leaseTTL = ttl
	}
}
