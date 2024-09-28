package detector_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/rafaeljusto/anicetus"
	"github.com/rafaeljusto/anicetus/detector"
)

func TestTokenBucketInMemory_IsThunderingHerd(t *testing.T) {
	tests := []struct {
		burst      int64
		interval   time.Duration
		cycles     int
		cycleSleep time.Duration
		want       func(cycle int) bool
	}{{
		burst:    1,
		interval: time.Second,
		cycles:   2,
		want: func(cycle int) bool {
			return cycle == 1
		},
	}, {
		burst:      4,
		interval:   500 * time.Millisecond,
		cycles:     6,
		cycleSleep: 100 * time.Millisecond,
		want: func(cycle int) bool {
			return cycle == 4
		},
	}}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("interval %s and burst %d", tt.interval, tt.burst), func(t *testing.T) {
			detector := detector.NewTokenBucketInMemory(
				detector.WithLimitersBurst(tt.burst),
				detector.WithLimitersInterval(tt.interval),
			)

			for i := 0; i < tt.cycles; i++ {
				t.Run("cycle"+strconv.Itoa(i), func(t *testing.T) {
					ok, err := detector.IsThunderingHerd(anicetus.Fingerprint("test"))
					if err != nil {
						t.Errorf("unexpected error: %v", err)
					}
					if want := tt.want(i); ok != want {
						t.Errorf("unexpected result: got %v, want %v", ok, want)
					}
					if tt.cycleSleep > 0 {
						time.Sleep(tt.cycleSleep)
					}
				})
			}
		})
	}
}
