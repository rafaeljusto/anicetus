package http

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/rafaeljusto/anicetus"
	"github.com/rafaeljusto/anicetus/detector"
	"github.com/rafaeljusto/anicetus/fingerprint"
	"github.com/rafaeljusto/anicetus/storage"
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
				detector.WithLimitersBurst(config.Detector.RequestsPerMinute),
				detector.WithLimitersInterval(time.Minute),
				detector.WithCoolDownInterval(config.Detector.CoolDown),
			),
			storage.NewInMemory(),
		),
	}

	resources.BackendClient = &http.Client{
		Timeout: config.Backend.Timeout,
	}

	return resources
}
