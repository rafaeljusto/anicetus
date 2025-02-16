package http

import (
	"log/slog"
	"net/http"

	"github.com/rafaeljusto/anicetus/v2"
	"github.com/rafaeljusto/anicetus/v2/fingerprint"
)

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
			if err := forwardRequest(w, r, config, resources); err != nil {
				httpLogger.Error("failed to forward request",
					slog.String("error", err.Error()),
				)
				w.WriteHeader(http.StatusInternalServerError)
			}
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

func loggerWrapper(logger *slog.Logger, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("request received",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)
		handler(w, r)
	}
}
