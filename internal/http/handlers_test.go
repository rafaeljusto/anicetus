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
