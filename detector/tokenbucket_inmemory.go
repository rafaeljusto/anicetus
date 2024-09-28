package detector

import (
	"time"

	"github.com/rafaeljusto/anicetus"
	"github.com/rafaeljusto/anicetus/internal/mapexp"
	"golang.org/x/time/rate"
)

var _ anicetus.Detector = &TokenBucketInMemory{}

// TokenBucketInMemoryOptions represents the options that can be used to
// configure a TokenBucketInMemory.
type TokenBucketInMemoryOptions struct {
	coolDownInterval time.Duration
	limitersBurst    int64
	limitersInterval time.Duration
}

// TokenBucketInMemoryOption is a helper function to configure the
// TokenBucketInMemory.
type TokenBucketInMemoryOption func(*TokenBucketInMemoryOptions)

// WithCoolDownInterval sets the cooldown interval for the TokenBucketInMemory.
func WithCoolDownInterval(interval time.Duration) TokenBucketInMemoryOption {
	return func(o *TokenBucketInMemoryOptions) {
		o.coolDownInterval = interval
	}
}

// WithLimitersBurst sets the burst for the limiters in the TokenBucketInMemory.
func WithLimitersBurst(burst int64) TokenBucketInMemoryOption {
	return func(o *TokenBucketInMemoryOptions) {
		o.limitersBurst = burst
	}
}

// WithLimitersInterval sets the interval for the limiters in the
// TokenBucketInMemory.
func WithLimitersInterval(interval time.Duration) TokenBucketInMemoryOption {
	return func(o *TokenBucketInMemoryOptions) {
		o.limitersInterval = interval
	}
}

// TokenBucketInMemory is a token bucket detector strategy that stores the state
// in memory.
type TokenBucketInMemory struct {
	cooldowns        *mapexp.Map[anicetus.Fingerprint, bool]
	limiters         *mapexp.Map[anicetus.Fingerprint, *rate.Limiter]
	limitersBurst    int64
	limitersInterval time.Duration
}

// NewTokenBucketInMemory creates a new token bucket detector strategy.
func NewTokenBucketInMemory(options ...TokenBucketInMemoryOption) *TokenBucketInMemory {
	o := TokenBucketInMemoryOptions{
		coolDownInterval: 5 * time.Minute,
		limitersBurst:    1000,
		limitersInterval: 1 * time.Minute,
	}
	for _, opt := range options {
		opt(&o)
	}

	return &TokenBucketInMemory{
		cooldowns:        mapexp.New[anicetus.Fingerprint, bool](o.coolDownInterval),
		limiters:         mapexp.New[anicetus.Fingerprint, *rate.Limiter](o.limitersInterval),
		limitersBurst:    o.limitersBurst,
		limitersInterval: o.limitersInterval,
	}
}

// CoolDown will cool down the fingerprint.
func (d *TokenBucketInMemory) CoolDown(fingerprint anicetus.Fingerprint) error {
	d.cooldowns.Set(fingerprint, true)
	return nil
}

// IsCoolDown checks if the fingerprint is in cooldown.
func (d *TokenBucketInMemory) IsCoolDown(fingerprint anicetus.Fingerprint) (bool, error) {
	cooldown, ok := d.cooldowns.Get(fingerprint)
	return cooldown && ok, nil
}

// IsThunderingHerd checks if the fingerprint is a thundering herd.
func (d *TokenBucketInMemory) IsThunderingHerd(fingerprint anicetus.Fingerprint) (bool, error) {
	limiter, ok := d.limiters.Get(fingerprint)
	if !ok {
		limiter = rate.NewLimiter(rate.Every(d.limitersInterval), int(d.limitersBurst))
		d.limiters.Set(fingerprint, limiter)
	}
	return !limiter.Allow(), nil
}
