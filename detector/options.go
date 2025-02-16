package detector

import (
	"log/slog"
	"time"
)

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

// Option is a helper function to configure the detector.
type Option func(*Options)

// WithLogger sets the logger to be used internally.
func WithLogger(logger *slog.Logger) Option {
	return func(o *Options) {
		o.logger = logger
	}
}

// TokenBucketOptions represents the options that can be used to configure a
// token bucket strategy.
type TokenBucketOptions struct {
	Options

	coolDownInterval time.Duration
	limitersBurst    int64
	limitersInterval time.Duration
}

// NewTokenBucketOptions creates a new TokenBucketOptions with default values.
func NewTokenBucketOptions() *TokenBucketOptions {
	return &TokenBucketOptions{
		Options: *NewOptions(),

		coolDownInterval: 5 * time.Minute,
		limitersBurst:    1000,
		limitersInterval: 1 * time.Minute,
	}
}

// CoolDownInterval returns the cooldown interval for the TokenBucketOptions.
func (o *TokenBucketOptions) CoolDownInterval() time.Duration {
	return o.coolDownInterval
}

// LimitersBurst returns the burst for the limiters in the TokenBucketOptions.
func (o *TokenBucketOptions) LimitersBurst() int64 {
	return o.limitersBurst
}

// LimitersInterval returns the interval for the limiters in the TokenBucketOptions.
func (o *TokenBucketOptions) LimitersInterval() time.Duration {
	return o.limitersInterval
}

// TokenBucketOption is a helper function to configure the TokenBucketOptions.
type TokenBucketOption func(*TokenBucketOptions)

// TokenBucketWithBasicOption sets the basic options for the TokenBucketOptions.
func TokenBucketWithBasicOption(options ...Option) TokenBucketOption {
	return func(o *TokenBucketOptions) {
		for _, opt := range options {
			opt(&o.Options)
		}
	}
}

// TokenBucketWithCoolDownInterval sets the cooldown interval for the
// TokenBucketOptions.
func TokenBucketWithCoolDownInterval(interval time.Duration) TokenBucketOption {
	return func(o *TokenBucketOptions) {
		o.coolDownInterval = interval
	}
}

// TokenBucketWithLimitersBurst sets the burst for the limiters in the
// TokenBucketOptions.
func TokenBucketWithLimitersBurst(burst int64) TokenBucketOption {
	return func(o *TokenBucketOptions) {
		o.limitersBurst = burst
	}
}

// TokenBucketWithLimitersInterval sets the interval for the limiters in the
// TokenBucketOptions.
func TokenBucketWithLimitersInterval(interval time.Duration) TokenBucketOption {
	return func(o *TokenBucketOptions) {
		o.limitersInterval = interval
	}
}
