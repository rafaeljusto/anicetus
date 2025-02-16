package detector

import (
	"context"
	"time"

	"github.com/rafaeljusto/anicetus/v2"
	"github.com/rafaeljusto/anicetus/v2/internal/mapexp"
	"github.com/rafaeljusto/anicetus/v2/internal/rate"
)

var _ anicetus.Detector = &TokenBucketInMemory{}

// TokenBucketInMemory is a token bucket detector strategy that stores the state
// in memory.
type TokenBucketInMemory struct {
	cooldowns        *mapexp.Map[anicetus.Fingerprint, bool]
	limiters         *mapexp.Map[anicetus.Fingerprint, *rate.Limiter]
	limitersBurst    int64
	limitersInterval time.Duration
}

// NewTokenBucketInMemory creates a new token bucket detector strategy.
func NewTokenBucketInMemory(options ...TokenBucketOption) *TokenBucketInMemory {
	o := NewTokenBucketOptions()
	for _, opt := range options {
		opt(o)
	}

	fullBucketPeriod := o.LimitersInterval() * time.Duration(o.limitersBurst)

	return &TokenBucketInMemory{
		cooldowns:        mapexp.New[anicetus.Fingerprint, bool](o.CoolDownInterval()),
		limiters:         mapexp.New[anicetus.Fingerprint, *rate.Limiter](fullBucketPeriod),
		limitersBurst:    o.LimitersBurst(),
		limitersInterval: o.LimitersInterval(),
	}
}

// CoolDown will cool down the fingerprint.
func (t *TokenBucketInMemory) CoolDown(_ context.Context, fingerprint anicetus.Fingerprint) error {
	t.cooldowns.Set(fingerprint, true)
	return nil
}

// IsCoolDown checks if the fingerprint is in cooldown.
func (t *TokenBucketInMemory) IsCoolDown(_ context.Context, fingerprint anicetus.Fingerprint) (bool, error) {
	cooldown, ok := t.cooldowns.Get(fingerprint)
	return cooldown && ok, nil
}

// IsThunderingHerd checks if the fingerprint is a thundering herd.
func (t *TokenBucketInMemory) IsThunderingHerd(_ context.Context, fingerprint anicetus.Fingerprint) (bool, error) {
	limiter, ok := t.limiters.Get(fingerprint)
	if !ok {
		limiter = rate.NewLimiter(rate.Every(t.limitersInterval), int(t.limitersBurst))
		t.limiters.Set(fingerprint, limiter)
	}
	return !limiter.Allow(), nil
}
