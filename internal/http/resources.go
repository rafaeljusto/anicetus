package http

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/rafaeljusto/anicetus/v3"
	"github.com/rafaeljusto/anicetus/v3/detector"
	"github.com/rafaeljusto/anicetus/v3/fingerprint"
	"github.com/rafaeljusto/anicetus/v3/storage"
)

// Resources stores the resources for the web server.
type Resources struct {
	Logger        *slog.Logger
	Anicetus      *anicetus.Anicetus[fingerprint.HTTPRequest]
	BackendClient *http.Client

	// detector is retained so its background goroutines can be stopped on Close.
	detector *detector.TokenBucketInMemory
}

// NewResources creates a new set of resources for the web server.
func NewResources(config *Config) *Resources {
	tokenBucket := detector.NewTokenBucketInMemory(
		detector.TokenBucketWithLimitersBurst(config.Detector.RequestsPerMinute),
		detector.TokenBucketWithLimitersInterval(time.Minute),
		detector.TokenBucketWithCoolDownInterval(config.Detector.CoolDown),
	)

	resources := &Resources{
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: config.LoggerLevel,
		})),
		Anicetus: anicetus.NewAnicetus[fingerprint.HTTPRequest](
			tokenBucket,
			storage.NewInMemory(),
		),
		detector: tokenBucket,
	}

	resources.BackendClient = &http.Client{
		Timeout: config.Backend.Timeout,
	}

	return resources
}

// Close releases the resources held by the web server, stopping the detector's
// background goroutines.
func (r *Resources) Close() error {
	return r.detector.Close()
}
