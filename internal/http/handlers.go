package http

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/rafaeljusto/anicetus/v3"
	"github.com/rafaeljusto/anicetus/v3/fingerprint"
)

// retryAfterSeconds is the Retry-After value sent to a blocked request that is
// not served, telling the client how long to back off before retrying.
const retryAfterSeconds = 1

// RegisterHandlers registers the handlers for the web server.
func RegisterHandlers(router *http.ServeMux, config *Config, resources *Resources) {
	router.HandleFunc("/", loggerWrapper(resources.Logger, anicetusHandler(config, resources)))
}

func anicetusHandler(config *Config, resources *Resources) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpLogger := resources.Logger.With(
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)

		if r.Method != http.MethodGet {
			// Only GET requests are subject to thundering herd control; anything
			// else is forwarded straight to the backend. Return afterwards so the
			// request is not evaluated and forwarded a second time.
			if err := forwardRequest(w, r, config, resources); err != nil {
				httpLogger.Error("failed to forward request",
					slog.String("error", err.Error()),
				)
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		fingerprint := fingerprint.NewHTTPRequest(r,
			fingerprint.WithHTTPRequestFields(config.Fingerprint.Fields...),
			fingerprint.WithHTTPRequestHeaders(config.Fingerprint.Headers...),
			fingerprint.WithHTTPRequestCookies(config.Fingerprint.Cookies...),
		)

		gatekeeperStatus, err := resources.Anicetus.Evaluate(r.Context(), fingerprint)
		if err != nil {
			httpLogger.Error("failed to analyze fingerprint",
				slog.String("error", err.Error()),
			)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// When configured to wait, hold a blocked request until the elected
		// request finishes (so it can be served from the warm cache) instead of
		// rejecting it immediately.
		if gatekeeperStatus == anicetus.StatusWait && config.Wait.Timeout > 0 {
			gatekeeperStatus = waitForGate(r.Context(), config, resources, fingerprint, httpLogger)
		}

		switch gatekeeperStatus {
		case anicetus.StatusFailed:
			w.WriteHeader(http.StatusInternalServerError)

		case anicetus.StatusProcess:
			httpLogger.With(slog.String("fingerprint", string(fingerprint.Fingerprint()))).
				Warn("thundering herd detected: processing single request")

			err := forwardRequest(w, r, config, resources,
				forwardRequestWithAnicetus(gatekeeperStatus, fingerprint.Fingerprint()),
				forwardRequestWithResponseHandler(func(*http.Response) error {
					return resources.Anicetus.RequestDone(r.Context(), fingerprint)
				}),
			)
			if err != nil {
				httpLogger.Error("failed to forward request",
					slog.String("error", err.Error()),
				)
				w.WriteHeader(http.StatusInternalServerError)

				if err := resources.Anicetus.Cleanup(r.Context(), fingerprint); err != nil {
					httpLogger.Error("failed to remove fingerprint",
						slog.String("error", err.Error()),
					)
				}
				return
			}

		case anicetus.StatusWait:
			// Either waiting is disabled or we waited out the timeout without the
			// elected request finishing. Ask the client to retry.
			w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds))
			w.WriteHeader(http.StatusServiceUnavailable)

		case anicetus.StatusOpenGates:
			err := forwardRequest(w, r, config, resources,
				forwardRequestWithAnicetus(gatekeeperStatus, fingerprint.Fingerprint()),
			)
			if err != nil {
				httpLogger.Error("failed to forward request",
					slog.String("error", err.Error()),
				)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	}
}

// waitForGate blocks until the request is no longer waiting behind the elected
// request -- either because that request finished (StatusOpenGates) or because
// its lease expired and this request was elected instead (StatusProcess) -- or
// until the configured wait timeout elapses. It returns the resolved status,
// which is StatusWait when the timeout is reached.
func waitForGate(
	ctx context.Context,
	config *Config,
	resources *Resources,
	f fingerprint.HTTPRequest,
	logger *slog.Logger,
) anicetus.Status {
	ctx, cancel := context.WithTimeout(ctx, config.Wait.Timeout)
	defer cancel()

	ticker := time.NewTicker(config.Wait.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return anicetus.StatusWait
		case <-ticker.C:
			status, err := resources.Anicetus.Evaluate(ctx, f)
			if err != nil {
				logger.Error("failed to re-evaluate fingerprint while waiting",
					slog.String("error", err.Error()),
				)
				return anicetus.StatusFailed
			}
			if status != anicetus.StatusWait {
				return status
			}
		}
	}
}

func loggerWrapper(logger *slog.Logger, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("request received",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)
		handler(w, r)
	}
}
