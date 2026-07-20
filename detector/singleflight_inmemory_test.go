package detector_test

import (
	"testing"
	"time"

	"github.com/rafaeljusto/anicetus/v3"
	"github.com/rafaeljusto/anicetus/v3/detector"
)

func TestSingleFlightInMemory_IsThunderingHerd(t *testing.T) {
	d := detector.NewSingleFlightInMemory()

	// A single-flight detector engages from the very first request, with no
	// burst to warm up first.
	for i := 0; i < 5; i++ {
		herd, err := d.IsThunderingHerd(t.Context(), anicetus.Fingerprint("test"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !herd {
			t.Errorf("cycle %d: expected thundering herd to always be reported", i)
		}
	}
}

func TestSingleFlightInMemory_CoolDown(t *testing.T) {
	d := detector.NewSingleFlightInMemory(
		detector.SingleFlightWithCoolDownInterval(time.Minute),
	)

	fingerprint := anicetus.Fingerprint("test")

	if cooldown, err := d.IsCoolDown(t.Context(), fingerprint); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if cooldown {
		t.Error("expected fingerprint to not be in cooldown before it is set")
	}

	if err := d.CoolDown(t.Context(), fingerprint); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cooldown, err := d.IsCoolDown(t.Context(), fingerprint); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if !cooldown {
		t.Error("expected fingerprint to be in cooldown after it is set")
	}
}
