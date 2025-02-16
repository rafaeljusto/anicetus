package anicetus_test

import (
	"context"
	"testing"

	"github.com/rafaeljusto/anicetus/v2"
)

func TestAnicetus_Evaluate(t *testing.T) {
	tests := []struct {
		name              string
		detector          anicetus.Detector
		gatekeeperStorage anicetus.GatekeeperStorage
		want              anicetus.Status
	}{{
		name: "it should allow requests when it is not a thundering herd",
		detector: fakeDetector{
			cooldown: false,
			anicetus: false,
		},
		gatekeeperStorage: &fakeGatekeeperStorage{
			exists:    false,
			processed: false,
		},
		want: anicetus.StatusOpenGates,
	}, {
		name: "it should allow the first request in the thundering herd",
		detector: fakeDetector{
			cooldown: false,
			anicetus: true,
		},
		gatekeeperStorage: &fakeGatekeeperStorage{
			exists:    false,
			processed: false,
		},
		want: anicetus.StatusProcess,
	}, {
		name: "it should block following requests in the thundering herd",
		detector: fakeDetector{
			cooldown: false,
			anicetus: true,
		},
		gatekeeperStorage: &fakeGatekeeperStorage{
			exists:    true,
			processed: false,
		},
		want: anicetus.StatusWait,
	}, {
		name: "it should allow all requests in the thundering herd once first is processed",
		detector: fakeDetector{
			cooldown: false,
			anicetus: true,
		},
		gatekeeperStorage: &fakeGatekeeperStorage{
			exists:    true,
			processed: true,
		},
		want: anicetus.StatusOpenGates,
	}, {
		name: "it should not check thundering herd if it is in cooldown",
		detector: fakeDetector{
			cooldown: true,
			anicetus: true,
		},
		gatekeeperStorage: &fakeGatekeeperStorage{
			exists:    false,
			processed: false,
		},
		want: anicetus.StatusOpenGates,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := anicetus.NewAnicetus[fakeFingerprinter](tt.detector, tt.gatekeeperStorage)

			status, err := th.Evaluate(t.Context(), fakeFingerprinter{})
			if err != nil {
				t.Errorf("unexpected error '%v'", err)
			}
			if status != tt.want {
				t.Errorf("unexpected status '%v', want '%v'", status, tt.want)
			}
		})
	}
}

func TestAnicetus_Evaluate_fullCycle(t *testing.T) {
	th := anicetus.NewAnicetus[fakeFingerprinter](fakeDetector{
		anicetus: true,
	}, &fakeGatekeeperStorage{})

	var fingerprinter fakeFingerprinter

	status, err := th.Evaluate(t.Context(), fingerprinter)
	if err != nil {
		t.Fatalf("unexpected error '%v'", err)
	}
	if status != anicetus.StatusProcess {
		t.Fatalf("unexpected status '%v', want '%v'", status, anicetus.StatusProcess)
	}

	if err := th.RequestDone(t.Context(), fingerprinter); err != nil {
		t.Fatalf("unexpected error '%v'", err)
	}

	status, err = th.Evaluate(t.Context(), fingerprinter)
	if err != nil {
		t.Fatalf("unexpected error '%v'", err)
	}
	if status != anicetus.StatusOpenGates {
		t.Fatalf("unexpected status '%v', want '%v'", status, anicetus.StatusOpenGates)
	}

	if err := th.Cleanup(t.Context(), fingerprinter); err != nil {
		t.Fatalf("unexpected error '%v'", err)
	}

	status, err = th.Evaluate(t.Context(), fingerprinter)
	if err != nil {
		t.Fatalf("unexpected error '%v'", err)
	}
	if status != anicetus.StatusProcess {
		t.Fatalf("unexpected status '%v', want '%v'", status, anicetus.StatusProcess)
	}
}

var _ anicetus.Fingerprinter = fakeFingerprinter{}
var _ anicetus.Detector = fakeDetector{}
var _ anicetus.GatekeeperStorage = &fakeGatekeeperStorage{}

// fakeFingerprinter is a fake implementation of Fingerprinter.
type fakeFingerprinter struct{}

func (f fakeFingerprinter) Fingerprint() anicetus.Fingerprint {
	return "fake"
}

// fakeDetector is a fake implementation of Detector.
type fakeDetector struct {
	cooldown bool
	anicetus bool
}

func (d fakeDetector) IsCoolDown(context.Context, anicetus.Fingerprint) (bool, error) {
	return d.cooldown, nil
}

func (d fakeDetector) CoolDown(context.Context, anicetus.Fingerprint) error {
	return nil
}

func (d fakeDetector) IsThunderingHerd(context.Context, anicetus.Fingerprint) (bool, error) {
	return d.anicetus, nil
}

// fakeGatekeeperStorage is a fake implementation of GatekeeperStorage.
type fakeGatekeeperStorage struct {
	exists    bool
	processed bool
}

func (gs fakeGatekeeperStorage) Exists(context.Context, anicetus.Fingerprint) (bool, error) {
	return gs.exists, nil
}

func (gs fakeGatekeeperStorage) Processed(context.Context, anicetus.Fingerprint) (bool, error) {
	return gs.processed, nil
}

func (gs *fakeGatekeeperStorage) Store(_ context.Context, _ anicetus.Fingerprint, processed bool) error {
	gs.exists = true
	gs.processed = processed
	return nil
}

func (gs *fakeGatekeeperStorage) Remove(context.Context, anicetus.Fingerprint) error {
	gs.exists = false
	gs.processed = false
	return nil
}
