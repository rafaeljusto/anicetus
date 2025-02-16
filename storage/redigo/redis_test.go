//go:build integration_tests
// +build integration_tests

package redigo_test

import (
	"context"
	"os"
	"testing"

	"github.com/gomodule/redigo/redis"
	"github.com/rafaeljusto/anicetus/v2"
	"github.com/rafaeljusto/anicetus/v2/storage/redigo"
)

const defaultRedisAddress = "localhost:6379"

func TestRedis_lifecycle(t *testing.T) {
	fingerprint := anicetus.Fingerprint("test")

	redisAddress := defaultRedisAddress
	if e := os.Getenv("REDIS_ADDRESS"); e != "" {
		redisAddress = e
	}

	redisPool := &redis.Pool{
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
	_, err = redisConn.Do("FLUSHDB")
	if err != nil {
		t.Fatalf("failed to flush redis database: %v", err)
	}

	storage := redigo.NewRedis(redisPool)
	if ok, err := storage.Exists(t.Context(), fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if ok {
		t.Error("unexpected fingerprint exists")
	}

	if ok, err := storage.Processed(t.Context(), fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if ok {
		t.Error("unexpected fingerprint processed")
	}

	if err := storage.Store(t.Context(), fingerprint, false); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ok, err := storage.Exists(t.Context(), fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if !ok {
		t.Error("fingerprint should exists")
	}

	if ok, err := storage.Processed(t.Context(), fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if ok {
		t.Error("fingerprint should not be processed")
	}

	if err := storage.Store(t.Context(), fingerprint, true); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ok, err := storage.Processed(t.Context(), fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if !ok {
		t.Error("fingerprint should be processed")
	}

	if err := storage.Remove(t.Context(), fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ok, err := storage.Exists(t.Context(), fingerprint); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if ok {
		t.Error("fingerprint should not exists")
	}
}
