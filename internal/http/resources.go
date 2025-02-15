package http

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/rafaeljusto/anicetus/v2"
	"github.com/rafaeljusto/anicetus/v2/detector"
	"github.com/rafaeljusto/anicetus/v2/fingerprint"
	"github.com/rafaeljusto/anicetus/v2/storage"
)

// Resources stores the resources for the web server.
type Resources struct {
	Logger        *slog.Logger
	Anicetus      *anicetus.Anicetus[fingerprint.HTTPRequest]
	BackendClient *http.Client
}

// NewResources creates a new set of resources for the web server.
func NewResources(config *Config) *Resources {
	resources := &Resources{
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: config.LoggerLevel,
		})),
		Anicetus: anicetus.NewAnicetus[fingerprint.HTTPRequest](
			detector.NewTokenBucketInMemory(
				detector.TokenBucketWithLimitersBurst(config.Detector.RequestsPerMinute),
				detector.TokenBucketWithLimitersInterval(time.Minute),
				detector.TokenBucketWithCoolDownInterval(config.Detector.CoolDown),
			),
			storage.NewInMemory(),
		),
	}

	resources.BackendClient = &http.Client{
		Timeout: config.Backend.Timeout,
	}

	return resources
}
