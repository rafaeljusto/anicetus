package redigo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/rafaeljusto/anicetus/v2"
	"github.com/rafaeljusto/anicetus/v2/storage"
)

var _ anicetus.GatekeeperStorage = &Redis{}

// keyPrefix namespaces the gatekeeper keys so they cannot collide with keys
// owned by other components sharing the same Redis database (e.g. the detector,
// which uses its own "anicetus:th:" and "anicetus:cooldown:" prefixes).
const keyPrefix = "anicetus:gk:"

// Redis is a redis storage for the fingerprints.
type Redis struct {
	pool     *redis.Pool
	logger   *slog.Logger
	leaseTTL time.Duration
}

// NewRedis creates a new redis storage.
func NewRedis(pool *redis.Pool, options ...storage.Option) *Redis {
	o := storage.NewOptions()
	for _, opt := range options {
		opt(o)
	}

	return &Redis{
		pool:     pool,
		logger:   o.Logger(),
		leaseTTL: o.LeaseTTL(),
	}
}

// Add atomically records the fingerprint as in-flight if it is not already
// present, returning true only for the caller that created the entry. It relies
// on Redis' atomic "SET key value NX PX ttl": NX makes the write conditional on
// absence (the election), while PX attaches the lease so a crashed elected
// request cannot block the fingerprint forever.
func (r *Redis) Add(ctx context.Context, fingerprint anicetus.Fingerprint) (bool, error) {
	conn, err := r.pool.GetContext(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get redis connection: %w", err)
	}
	defer r.close(conn)

	result, err := redis.String(conn.Do("SET", key(fingerprint), 0, "NX", "PX", r.leaseMillis()))
	if errors.Is(err, redis.ErrNil) {
		// NX prevented the write: the fingerprint was already elected.
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to set redis key: %w", err)
	}
	return result == "OK", nil
}

// Processed checks if the fingerprint was processed.
func (r *Redis) Processed(ctx context.Context, fingerprint anicetus.Fingerprint) (bool, error) {
	conn, err := r.pool.GetContext(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get redis connection: %w", err)
	}
	defer r.close(conn)

	result, err := redis.Bool(conn.Do("GET", key(fingerprint)))
	if errors.Is(err, redis.ErrNil) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to get redis key: %w", err)
	}
	return result, nil
}

// Store stores the fingerprint in the storage, refreshing its lease.
func (r *Redis) Store(ctx context.Context, fingerprint anicetus.Fingerprint, processed bool) error {
	conn, err := r.pool.GetContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to get redis connection: %w", err)
	}
	defer r.close(conn)

	result, err := redis.String(conn.Do("SET", key(fingerprint), boolToInt(processed), "PX", r.leaseMillis()))
	if err != nil {
		return fmt.Errorf("failed to set redis key: %w", err)
	}
	if result != "OK" {
		return fmt.Errorf("failed to set redis key")
	}
	return nil
}

// Remove removes the fingerprint from the storage.
func (r *Redis) Remove(ctx context.Context, fingerprint anicetus.Fingerprint) error {
	conn, err := r.pool.GetContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to get redis connection: %w", err)
	}
	defer r.close(conn)

	_, err = redis.Int(conn.Do("DEL", key(fingerprint)))
	if err != nil {
		return fmt.Errorf("failed to delete redis key: %w", err)
	}
	return nil
}

// leaseMillis returns the lease duration in milliseconds, clamped to at least 1
// so the PX argument is always valid.
func (r *Redis) leaseMillis() int64 {
	if ms := r.leaseTTL.Milliseconds(); ms > 0 {
		return ms
	}
	return 1
}

// close returns the connection to the pool, logging any error.
func (r *Redis) close(conn redis.Conn) {
	if err := conn.Close(); err != nil {
		if r.logger != nil {
			r.logger.Error("failed to close redis connection", slog.String("error", err.Error()))
		}
	}
}

// key namespaces a fingerprint into the gatekeeper key space.
func key(fingerprint anicetus.Fingerprint) string {
	return keyPrefix + string(fingerprint)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
