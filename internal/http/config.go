package http

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rafaeljusto/anicetus/v3/fingerprint"
)

// Config stores the configuration for the application.
type Config struct {
	Port        int64
	LoggerLevel slog.Level
	Fingerprint struct {
		Fields  []fingerprint.HTTPRequestField
		Headers []string
		Cookies []string
	}
	Detector struct {
		RequestsPerMinute int64
		CoolDown          time.Duration
	}
	Backend struct {
		Timeout time.Duration
		Address *url.URL
	}
	Wait struct {
		// Timeout is how long a blocked request is held waiting for the elected
		// request to finish before giving up. Zero means do not wait: reply
		// immediately with 503 and a Retry-After header.
		Timeout time.Duration
		// PollInterval is how often the gatekeeper is re-checked while waiting.
		PollInterval time.Duration
	}
}

// ParseFromEnvs parses the configuration from environment variables.
func ParseFromEnvs() (*Config, error) {
	var config Config
	var errs error
	var err error

	if portStr := os.Getenv("ANICETUS_PORT"); portStr != "" {
		config.Port, err = strconv.ParseInt(portStr, 10, 64)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to parse ANICETUS_PORT: %w", err))
		}
	}

	loggerLevel := slog.LevelInfo
	if loggerLevelStr := os.Getenv("ANICETUS_LOG_LEVEL"); loggerLevelStr != "" {
		if err = loggerLevel.UnmarshalText([]byte(loggerLevelStr)); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to parse ANICETUS_LOG_LEVEL: %w", err))
		}
	}
	config.LoggerLevel = loggerLevel

	fingerprintFields := []fingerprint.HTTPRequestField{
		fingerprint.HTTPRequestFieldProto,
		fingerprint.HTTPRequestFieldMethod,
		fingerprint.HTTPRequestFieldHost,
		fingerprint.HTTPRequestFieldPath,
		fingerprint.HTTPRequestFieldQuery,
	}
	if fingerprintFieldsStr := os.Getenv("ANICETUS_FINGERPRINT_FIELDS"); fingerprintFieldsStr != "" {
		fingerprintFields = fingerprintFields[:0]
		for _, fieldStr := range strings.Split(fingerprintFieldsStr, ",") {
			field, err := fingerprint.ParseHTTPRequestField(fieldStr)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to parse ANICETUS_FINGERPRINT_FIELDS: %w", err))
			}
			fingerprintFields = append(fingerprintFields, field)
		}
	}
	config.Fingerprint.Fields = fingerprintFields

	if fingerprintHeadersStr := os.Getenv("ANICETUS_FINGERPRINT_HEADERS"); fingerprintHeadersStr != "" {
		config.Fingerprint.Headers = strings.Split(fingerprintHeadersStr, ",")

		var i int
		for _, header := range config.Fingerprint.Headers {
			header = strings.TrimSpace(header)
			if header != "" {
				config.Fingerprint.Headers[i] = header
				i++
			}
		}
		config.Fingerprint.Headers = config.Fingerprint.Headers[:i]
	}

	if fingerprintCookiesStr := os.Getenv("ANICETUS_FINGERPRINT_COOKIES"); fingerprintCookiesStr != "" {
		config.Fingerprint.Cookies = strings.Split(fingerprintCookiesStr, ",")

		var i int
		for _, cookie := range config.Fingerprint.Cookies {
			cookie = strings.TrimSpace(cookie)
			if cookie != "" {
				config.Fingerprint.Cookies[i] = cookie
				i++
			}
		}
		config.Fingerprint.Cookies = config.Fingerprint.Cookies[:i]
	}

	config.Detector.RequestsPerMinute = 1000
	if requestsPerMinuteStr := os.Getenv("ANICETUS_DETECTOR_REQUESTS_PER_MINUTE"); requestsPerMinuteStr != "" {
		config.Detector.RequestsPerMinute, err = strconv.ParseInt(requestsPerMinuteStr, 10, 64)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to parse ANICETUS_DETECTOR_REQUESTS_PER_MINUTE: %w", err))
		}
	}

	config.Detector.CoolDown = 10 * time.Minute
	if coolDownStr := os.Getenv("ANICETUS_DETECTOR_COOLDOWN"); coolDownStr != "" {
		config.Detector.CoolDown, err = time.ParseDuration(coolDownStr)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to parse ANICETUS_DETECTOR_COOLDOWN: %w", err))
		}
	}

	timeout := time.Minute
	if timeoutStr := os.Getenv("ANICETUS_BACKEND_TIMEOUT"); timeoutStr != "" {
		timeout, err = time.ParseDuration(timeoutStr)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to parse ANICETUS_BACKEND_TIMEOUT: %w", err))
		}
	}
	config.Backend.Timeout = timeout

	if addressStr := os.Getenv("ANICETUS_BACKEND_ADDRESS"); addressStr == "" {
		errs = errors.Join(errs, fmt.Errorf("ANICETUS_BACKEND_ADDRESS is required"))
	} else if config.Backend.Address, err = url.Parse(addressStr); err != nil {
		errs = errors.Join(errs, fmt.Errorf("failed to parse ANICETUS_BACKEND_ADDRESS: %w", err))
	}

	// Zero disables waiting: blocked requests get an immediate 503 + Retry-After.
	if waitTimeoutStr := os.Getenv("ANICETUS_WAIT_TIMEOUT"); waitTimeoutStr != "" {
		config.Wait.Timeout, err = time.ParseDuration(waitTimeoutStr)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to parse ANICETUS_WAIT_TIMEOUT: %w", err))
		}
	}

	config.Wait.PollInterval = 50 * time.Millisecond
	if pollIntervalStr := os.Getenv("ANICETUS_WAIT_POLL_INTERVAL"); pollIntervalStr != "" {
		config.Wait.PollInterval, err = time.ParseDuration(pollIntervalStr)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to parse ANICETUS_WAIT_POLL_INTERVAL: %w", err))
		}
	}
	if config.Wait.Timeout > 0 && config.Wait.PollInterval <= 0 {
		errs = errors.Join(errs, fmt.Errorf("ANICETUS_WAIT_POLL_INTERVAL must be greater than zero"))
	}

	if errs != nil {
		return nil, errs
	}
	return &config, nil
}
