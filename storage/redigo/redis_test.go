//go:build integration_tests
// +build integration_tests

package redigo_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/rafaeljusto/anicetus/v3"
	"github.com/rafaeljusto/anicetus/v3/storage"
	"github.com/rafaeljusto/anicetus/v3/storage/redigo"
)

const defaultRedisAddress = "localhost:6379"

func newRedisPool(t *testing.T) *redis.Pool {
	t.Helper()

	redisAddress := defaultRedisAddress
	if e := os.Getenv("REDIS_ADDRESS"); e != "" {
		redisAddress = e
	}

	redisPool := &redis.Pool{
		MaxIdle:     10,
		MaxActive:   100,
		IdleTimeout: 5 * time.Minute,
		DialContext: func(ctx context.Context) (redis.Conn, error) {
			return redis.DialContext(ctx, "tcp", redisAddress)
		},
	}

	redisConn, err := redisPool.GetContext(t.Context())
	if err != nil {
		t.Fatalf("failed to get redis connection: %v", err)
	}
	defer func() {
		if err := redisConn.Close(); err != nil {
			t.Errorf("failed to close redis connection: %v", err)
		}
	}()
	if _, err := redisConn.Do("FLUSHDB"); err != nil {
		t.Fatalf("failed to flush redis database: %v", err)
	}

	return redisPool
}

func TestRedis_lifecycle(t *testing.T) {
	fingerprint := anicetus.Fingerprint("test")

	s := redigo.NewRedis(newRedisPool(t))

	if elected, err := s.Add(t.Context(), fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if !elected {
		t.Error("first Add should elect the fingerprint")
	}

	if elected, err := s.Add(t.Context(), fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if elected {
		t.Error("second Add should not elect the fingerprint")
	}

	if ok, err := s.Processed(t.Context(), fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if ok {
		t.Error("fingerprint should not be processed yet")
	}

	if err := s.Store(t.Context(), fingerprint, true); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ok, err := s.Processed(t.Context(), fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if !ok {
		t.Error("fingerprint should be processed")
	}

	if err := s.Remove(t.Context(), fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ok, err := s.Processed(t.Context(), fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if ok {
		t.Error("fingerprint should not be processed after removal")
	}

	if elected, err := s.Add(t.Context(), fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if !elected {
		t.Error("Add after Remove should elect the fingerprint again")
	}
}

// TestRedis_Add_leaseExpiry ensures the lease attached by Add releases the
// fingerprint once it expires, so a crashed elected request cannot block it
// forever.
func TestRedis_Add_leaseExpiry(t *testing.T) {
	// The lease must comfortably exceed the round-trip of a couple of commands
	// so the "within lease" Add reliably observes the key, while remaining short
	// enough to keep the test fast once we wait it out.
	lease := 2 * time.Second

	s := redigo.NewRedis(newRedisPool(t), storage.WithLeaseTTL(lease))
	fingerprint := anicetus.Fingerprint("test")

	if elected, err := s.Add(t.Context(), fingerprint); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if !elected {
		t.Fatal("first Add should elect the fingerprint")
	}

	if elected, err := s.Add(t.Context(), fingerprint); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if elected {
		t.Fatal("Add within the lease should not elect the fingerprint")
	}

	time.Sleep(2 * lease)

	if elected, err := s.Add(t.Context(), fingerprint); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if !elected {
		t.Error("Add after the lease expired should elect the fingerprint again")
	}
}
