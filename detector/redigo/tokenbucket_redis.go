package redigo

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/rafaeljusto/anicetus/v2"
	"github.com/rafaeljusto/anicetus/v2/detector"
)

var (
	_ anicetus.Detector = &TokenBucketRedis{}

	tokenBucketScript = redis.NewScript(1, `
-- Token Bucket rate limiter
-- KEYS[1]: The Redis key for storing the token bucket
-- ARGV[1]: Maximum capacity of the bucket (max_tokens)
-- ARGV[2]: Refill rate per second (tokens_per_second)

local key = KEYS[1]
local max_tokens = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local requested_tokens = 1
local current_time = redis.call("TIME")
local now = tonumber(current_time[1]) + tonumber(current_time[2]) / 1000000

-- Fetch the stored bucket data
local bucket = redis.call("HMGET", key, "tokens", "last_refreshed")
local tokens = tonumber(bucket[1]) or max_tokens
local last_refreshed = tonumber(bucket[2]) or now

-- Refill the tokens based on elapsed time
local elapsed_time = now - last_refreshed
local new_tokens = math.min(max_tokens, tokens + (elapsed_time * refill_rate))

-- Check if we have enough tokens
if new_tokens >= requested_tokens then
  new_tokens = new_tokens - requested_tokens

  local hmset_result = redis.call("HMSET", key, "tokens", new_tokens, "last_refreshed", now)
  if not hmset_result then
    redis.log(redis.LOG_NOTICE, "anicetus: failed to update token bucket for key: " .. key)
  end

  redis.call("EXPIRE", key, math.ceil(max_tokens / refill_rate))
  return 1 -- Allowed

else
  -- apply penalty for thundering herd
  local hmset_result = redis.call("HMSET", key, "tokens", 0, "last_refreshed", now)
  if not hmset_result then
    redis.log(redis.LOG_NOTICE, "anicetus: failed to update token bucket for key: " .. key)
    return 1 -- Allowed
  end

  return 0 -- Thundering herd
end
`)
)

// TokenBucketRedis is a token bucket detector strategy that stores the state in
// Redis.
type TokenBucketRedis struct {
	pool             *redis.Pool
	coolDownInterval time.Duration
	limitersBurst    int64
	limitersInterval time.Duration
	logger           *slog.Logger
}

// NewTokenBucketRedis creates a new token bucket detector strategy.
func NewTokenBucketRedis(pool *redis.Pool, options ...detector.TokenBucketOption) *TokenBucketRedis {
	o := detector.NewTokenBucketOptions()
	for _, opt := range options {
		opt(o)
	}

	return &TokenBucketRedis{
		pool:             pool,
		coolDownInterval: o.CoolDownInterval(),
		limitersBurst:    o.LimitersBurst(),
		limitersInterval: o.LimitersInterval(),
		logger:           o.Logger(),
	}
}

// CoolDown will cool down the fingerprint.
func (t *TokenBucketRedis) CoolDown(ctx context.Context, fingerprint anicetus.Fingerprint) error {
	conn, err := t.pool.GetContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to get redis connection: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			if t.logger != nil {
				t.logger.Error("failed to close redis connection", slog.String("error", err.Error()))
			}
		}
	}()

	result, err := redis.Int(conn.Do("SET", addKeyPrefix(fingerprint, modeCoolDown), 1,
		"EX", t.coolDownInterval.Seconds(),
	))
	if err != nil {
		return fmt.Errorf("failed to set redis key: %w", err)
	}
	if result != 1 {
		return fmt.Errorf("failed to set redis key")
	}
	return nil
}

// IsCoolDown checks if the fingerprint is in cooldown.
func (t *TokenBucketRedis) IsCoolDown(ctx context.Context, fingerprint anicetus.Fingerprint) (bool, error) {
	conn, err := t.pool.GetContext(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get redis connection: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			if t.logger != nil {
				t.logger.Error("failed to close redis connection", slog.String("error", err.Error()))
			}
		}
	}()

	result, err := redis.Int(conn.Do("EXISTS", addKeyPrefix(fingerprint, modeCoolDown)))
	if err != nil {
		return false, fmt.Errorf("failed to check redis key: %w", err)
	}
	return result == 1, nil
}

// IsThunderingHerd checks if the fingerprint is a thundering herd.
func (t *TokenBucketRedis) IsThunderingHerd(ctx context.Context, fingerprint anicetus.Fingerprint) (bool, error) {
	conn, err := t.pool.GetContext(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get redis connection: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			if t.logger != nil {
				t.logger.Error("failed to close redis connection", slog.String("error", err.Error()))
			}
		}
	}()

	allow, err := redis.Bool(tokenBucketScript.DoContext(ctx, conn, addKeyPrefix(fingerprint, modeThunderingHerd),
		t.limitersBurst,                // max tokens
		1/t.limitersInterval.Seconds(), // refill rate
	))
	if err != nil {
		return false, fmt.Errorf("failed to execute redis lua script: %w", err)
	}
	return !allow, nil
}

// mode is used to set the correct redis scope for the keys.
type mode string

const (
	// modeCoolDown is the mode used to check if the fingerprint is in cooldown.
	modeCoolDown mode = "cooldown"
	// modeThunderingHerd is the mode used to check if the fingerprint is a
	// thundering herd.
	modeThunderingHerd mode = "th"
)

// addKeyPrefix adds the key prefix to the fingerprint to correctly set the
// scope.
func addKeyPrefix(fingerprint anicetus.Fingerprint, m mode) string {
	return fmt.Sprintf("anicetus:%s:%s", m, fingerprint)
}
