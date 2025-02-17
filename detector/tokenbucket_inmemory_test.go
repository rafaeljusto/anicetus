package detector_test

import (
	"fmt"
	"slices"
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
		interval: 500 * time.Millisecond,
		cycles:   7,
		cycleSleep: func(cycle int) time.Duration {
			if cycle == 6 {
				// sleep longer to allow populating 1 token and avoid thundering herd
				return 500 * time.Millisecond
			}
			return 100 * time.Millisecond
		},
		want: func(cycle int) bool {
			return slices.Contains([]int{5, 6}, cycle)
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
