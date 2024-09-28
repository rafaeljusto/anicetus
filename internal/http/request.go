package http

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/rafaeljusto/anicetus"
)

type forwardRequestOptions struct {
	responseHandler func(*http.Response) error
	anicetus        struct {
		status      anicetus.Status
		fingerprint anicetus.Fingerprint
	}
}

type forwardRequestOption func(*forwardRequestOptions)

func forwardRequestWithResponseHandler(handler func(*http.Response) error) forwardRequestOption {
	return func(opts *forwardRequestOptions) {
		opts.responseHandler = handler
	}
}

func forwardRequestWithAnicetus(
	status anicetus.Status,
	fingerprint anicetus.Fingerprint,
) forwardRequestOption {
	return func(opts *forwardRequestOptions) {
		opts.anicetus.status = status
		opts.anicetus.fingerprint = fingerprint
	}
}

// forwardRequest forwards the request to the backend and writes the response
// back to the caller.
func forwardRequest(
	w http.ResponseWriter,
	r *http.Request,
	config *Config,
	resources *Resources,
	optFuncs ...forwardRequestOption,
) error {
	opts := &forwardRequestOptions{
		responseHandler: func(*http.Response) error {
			return nil
		},
	}
	for _, optFunc := range optFuncs {
		optFunc(opts)
	}

	r.URL.Host = config.Backend.Address.Host
	r.URL.Scheme = config.Backend.Address.Scheme

	req, err := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), r.Body)
	if err != nil {
		return fmt.Errorf("failed to create forwarded request: %w", err)
	}

	req.Header = r.Header
	req.Header.Add("X-Forwarded-For", r.RemoteAddr)
	if opts.anicetus.status != anicetus.StatusNone {
		req.Header.Set("Anicetus-Status", opts.anicetus.status.String())
		req.Header.Set("Anicetus-Fingerprint", opts.anicetus.fingerprint.String())
	}
	req.Host = r.Host

	response, err := resources.BackendClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute forwarded request: %w", err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			resources.Logger.Error("failed to close forwarded response body",
				slog.String("error", err.Error()),
			)
		}
	}()

	if err := opts.responseHandler(response); err != nil {
		return fmt.Errorf("failed to handle forwarded response: %w", err)
	}

	for key, values := range response.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(response.StatusCode)
	if _, err := io.Copy(w, response.Body); err != nil {
		return fmt.Errorf("failed to copy forwarded response body: %w", err)
	}

	return nil
}
