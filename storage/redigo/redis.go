package redigo

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gomodule/redigo/redis"
	"github.com/rafaeljusto/anicetus/v2"
	"github.com/rafaeljusto/anicetus/v2/storage"
)

var _ anicetus.GatekeeperStorage = &Redis{}

// Redis is a redis storage for the fingerprints.
type Redis struct {
	pool   *redis.Pool
	logger *slog.Logger
}

// NewRedis creates a new redis storage.
func NewRedis(pool *redis.Pool, options ...storage.Option) *Redis {
	o := storage.NewOptions()
	for _, opt := range options {
		opt(o)
	}

	return &Redis{
		pool:   pool,
		logger: o.Logger(),
	}
}

// Exists checks if the fingerprint exists in the storage.
func (r *Redis) Exists(ctx context.Context, fingerprint anicetus.Fingerprint) (bool, error) {
	conn, err := r.pool.GetContext(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get redis connection: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			if r.logger != nil {
				r.logger.Error("failed to close redis connection", slog.String("error", err.Error()))
			}
		}
	}()

	result, err := redis.Int(conn.Do("EXISTS", fingerprint))
	if err != nil {
		return false, fmt.Errorf("failed to check redis key: %w", err)
	}
	return result == 1, nil
}

// Processed checks if the fingerprint was processed.
func (r *Redis) Processed(ctx context.Context, fingerprint anicetus.Fingerprint) (bool, error) {
	conn, err := r.pool.GetContext(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get redis connection: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			if r.logger != nil {
				r.logger.Error("failed to close redis connection", slog.String("error", err.Error()))
			}
		}
	}()

	result, err := redis.Bool(conn.Do("GET", fingerprint))
	if err == redis.ErrNil {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to get redis key: %w", err)
	}
	return result, nil
}

// Store stores the fingerprint in the storage.
func (r *Redis) Store(ctx context.Context, fingerprint anicetus.Fingerprint, processed bool) error {
	conn, err := r.pool.GetContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to get redis connection: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			if r.logger != nil {
				r.logger.Error("failed to close redis connection", slog.String("error", err.Error()))
			}
		}
	}()

	result, err := redis.String(conn.Do("SET", fingerprint, boolToInt(processed)))
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
	defer func() {
		if err := conn.Close(); err != nil {
			if r.logger != nil {
				r.logger.Error("failed to close redis connection", slog.String("error", err.Error()))
			}
		}
	}()

	_, err = redis.Int(conn.Do("DEL", fingerprint))
	if err != nil {
		return fmt.Errorf("failed to delete redis key: %w", err)
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
