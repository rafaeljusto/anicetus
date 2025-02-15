//go:build integration_tests
// +build integration_tests

package redigo_test

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/rafaeljusto/anicetus/v2"
	"github.com/rafaeljusto/anicetus/v2/detector"
	"github.com/rafaeljusto/anicetus/v2/detector/redigo"
)

const defaultRedisAddress = "localhost:6379"

func TestTokenBucketRedis_IsThunderingHerd(t *testing.T) {
	tests := []struct {
		burst      int64
		interval   time.Duration
		cycles     int
		cycleSleep func(cycle int) time.Duration
		want       func(cycle int) bool
	}{{
		burst:    1,
		interval: time.Second,
		cycles:   2,
		want: func(cycle int) bool {
			return cycle == 2
		},
	}, {
		burst:    4,
		interval: 500 * time.Millisecond,
		cycles:   6,
		cycleSleep: func(cycle int) time.Duration {
			if cycle == 6 {
				// sleep longer to allow populating 1 token and avoid thundering herd
				return 500 * time.Millisecond
			}
			return 100 * time.Millisecond
		},
		want: func(cycle int) bool {
			// this may fail if the I/O with Redis is too slow, as the filling rate
			// will give a chance for cycle 5
			return slices.Contains([]int{5, 6}, cycle)
		},
	}}

	redisAddress := defaultRedisAddress
	if e := os.Getenv("REDIS_ADDRESS"); e != "" {
		redisAddress = e
	}

	redisPool := &redis.Pool{
		DialContext: func(ctx context.Context) (redis.Conn, error) {
			return redis.DialContext(ctx, "tcp", redisAddress)
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("interval %s and burst %d", tt.interval, tt.burst), func(t *testing.T) {
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

			detector := redigo.NewTokenBucketRedis(
				redisPool,
				detector.TokenBucketWithLimitersBurst(tt.burst),
				detector.TokenBucketWithLimitersInterval(tt.interval),
			)

			for i := 1; i <= tt.cycles; i++ {
				t.Run("cycle"+strconv.Itoa(i), func(t *testing.T) {
					ok, err := detector.IsThunderingHerd(t.Context(), anicetus.Fingerprint("test"))
					if err != nil {
						t.Errorf("unexpected error: %v", err)
					}
					if want := tt.want(i); ok != want {
						t.Errorf("unexpected result: got %v, want %v", ok, want)
					}
					if tt.cycleSleep != nil {
						if sleep := tt.cycleSleep(i); sleep > 0 {
							time.Sleep(sleep)
						}
					}
				})
			}
		})
	}
}
