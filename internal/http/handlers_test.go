package http

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rafaeljusto/anicetus/v3"
	"github.com/rafaeljusto/anicetus/v3/fingerprint"
)

// newTestHandler wires a handler in front of a backend that counts how many
// times it is hit.
func newTestHandler(t *testing.T) (http.HandlerFunc, *atomic.Int64) {
	t.Helper()

	var backendHits atomic.Int64
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		backendHits.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	}))
	t.Cleanup(backend.Close)

	backendURL, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatalf("failed to parse backend url: %v", err)
	}

	config := &Config{}
	config.LoggerLevel = slog.LevelError
	config.Detector.RequestsPerMinute = 1000
	config.Detector.CoolDown = time.Minute
	config.Backend.Timeout = time.Minute
	config.Backend.Address = backendURL
	config.Fingerprint.Fields = []fingerprint.HTTPRequestField{
		fingerprint.HTTPRequestFieldMethod,
		fingerprint.HTTPRequestFieldPath,
	}

	resources := NewResources(config)
	t.Cleanup(func() { _ = resources.Close() })

	return anicetusHandler(config, resources), &backendHits
}

// blockingBackend is a backend that blocks the elected (StatusProcess) request
// until released, letting a test hold the gate open while another request with
// the same fingerprint is evaluated.
type blockingBackend struct {
	entered chan struct{}
	release chan struct{}
}

// newWaitTestHandler wires a handler in front of a blocking backend with the
// given wait timeout. The detector uses a burst of one so the first request is
// allowed through and drains the bucket, and every subsequent request with the
// same fingerprint is treated as a thundering herd.
func newWaitTestHandler(t *testing.T, waitTimeout time.Duration) (http.HandlerFunc, *blockingBackend) {
	t.Helper()

	bb := &blockingBackend{
		entered: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Anicetus-Status") == anicetus.StatusProcess.String() {
			bb.entered <- struct{}{}
			<-bb.release
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	}))
	t.Cleanup(backend.Close)

	backendURL, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatalf("failed to parse backend url: %v", err)
	}

	config := &Config{}
	config.LoggerLevel = slog.LevelError
	config.Detector.RequestsPerMinute = 1
	config.Detector.CoolDown = time.Minute
	config.Backend.Timeout = time.Minute
	config.Backend.Address = backendURL
	config.Fingerprint.Fields = []fingerprint.HTTPRequestField{
		fingerprint.HTTPRequestFieldMethod,
		fingerprint.HTTPRequestFieldPath,
	}
	config.Wait.Timeout = waitTimeout
	config.Wait.PollInterval = 10 * time.Millisecond

	resources := NewResources(config)
	t.Cleanup(func() { _ = resources.Close() })

	return anicetusHandler(config, resources), bb
}

// get issues a GET for the shared fingerprint and returns the response recorder.
func get(handler http.HandlerFunc) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	handler(rr, httptest.NewRequest(http.MethodGet, "/resource", nil))
	return rr
}

// TestAnicetusHandler_waitDisabledReturns503 checks that with waiting disabled a
// blocked request is rejected immediately with 503 and a Retry-After header.
func TestAnicetusHandler_waitDisabledReturns503(t *testing.T) {
	handler, bb := newWaitTestHandler(t, 0)

	// Drain the bucket so the following requests are treated as a herd.
	get(handler)

	// The elected request holds the gate open inside the backend.
	electedDone := make(chan struct{})
	go func() {
		defer close(electedDone)
		get(handler)
	}()
	<-bb.entered

	rr := get(handler)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rr.Code)
	}
	if rr.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header to be set")
	}

	close(bb.release)
	<-electedDone
}

// TestAnicetusHandler_waitHoldsThenForwards checks that with waiting enabled a
// blocked request is held and then forwarded once the elected request finishes.
func TestAnicetusHandler_waitHoldsThenForwards(t *testing.T) {
	handler, bb := newWaitTestHandler(t, 2*time.Second)

	get(handler) // drain the bucket

	electedDone := make(chan struct{})
	go func() {
		defer close(electedDone)
		get(handler)
	}()
	<-bb.entered

	waiterDone := make(chan int, 1)
	go func() {
		waiterDone <- get(handler).Code
	}()

	// Release the elected request; the waiter must then be forwarded.
	close(bb.release)
	<-electedDone

	select {
	case code := <-waiterDone:
		if code != http.StatusOK {
			t.Errorf("expected waiting request to be forwarded with %d, got %d", http.StatusOK, code)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("waiting request did not complete in time")
	}
}

// TestAnicetusHandler_forwardsNonGetOnce is the regression test for the missing
// return that caused non-GET requests to be forwarded to the backend twice.
func TestAnicetusHandler_forwardsNonGetOnce(t *testing.T) {
	handler, backendHits := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/resource", strings.NewReader("body"))
	rr := httptest.NewRecorder()
	handler(rr, req)

	if got := backendHits.Load(); got != 1 {
		t.Errorf("expected backend to be hit once for POST, got %d", got)
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

// TestAnicetusHandler_forwardsGetOnce guards the normal (open gates) path so the
// return added for non-GET requests does not accidentally short-circuit GETs.
func TestAnicetusHandler_forwardsGetOnce(t *testing.T) {
	handler, backendHits := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if got := backendHits.Load(); got != 1 {
		t.Errorf("expected backend to be hit once for GET, got %d", got)
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}
