package storage

import "log/slog"

// Options provides all the available options.
type Options struct {
	// logger to be used internally.
	logger *slog.Logger
}

// NewOptions creates a new Options with default values.
func NewOptions() *Options {
	return &Options{}
}

// Logger returns the logger to be used internally.
func (o *Options) Logger() *slog.Logger {
	return o.logger
}

// Option is a helper function to configure the storage.
type Option func(*Options)

// WithLogger sets the logger to be used internally.
func WithLogger(logger *slog.Logger) Option {
	return func(o *Options) {
		o.logger = logger
	}
}
