package detector_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/rafaeljusto/anicetus/v2"
	"github.com/rafaeljusto/anicetus/v2/detector"
)

func TestTokenBucketInMemory_IsThunderingHerd(t *testing.T) {
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
		interval: 2 * time.Second,
		cycles:   6,
		cycleSleep: func(cycle int) time.Duration {
			if cycle == 5 {
				// wait more than one refill period so a single token is restored,
				// allowing cycle 6 through again
				return 2100 * time.Millisecond
			}
			// negligible refill while draining the bucket
			return 10 * time.Millisecond
		},
		want: func(cycle int) bool {
			// cycles 1-4 drain the bucket, cycle 5 finds it empty (thundering
			// herd), and the refill wait lets cycle 6 through again
			return cycle == 5
		},
	}}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("interval %s and burst %d", tt.interval, tt.burst), func(t *testing.T) {
			detector := detector.NewTokenBucketInMemory(
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
