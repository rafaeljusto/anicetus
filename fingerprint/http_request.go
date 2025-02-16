package fingerprint

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rafaeljusto/anicetus/v2"
)

var _ anicetus.Fingerprinter = &HTTPRequest{}

// HTTPRequestField represents the fields that can be used to fingerprint a web
// request.
type HTTPRequestField int

// List of fields that can be used to fingerprint a web request.
const (
	HTTPRequestFieldProto HTTPRequestField = 1 << iota
	HTTPRequestFieldMethod
	HTTPRequestFieldScheme
	HTTPRequestFieldHost
	HTTPRequestFieldPath
	HTTPRequestFieldQuery
	HTTPRequestFieldBody
)

// ParseHTTPRequestField parses a string into a HTTPRequestField.
func ParseHTTPRequestField(s string) (HTTPRequestField, error) {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)

	switch s {
	case "proto":
		return HTTPRequestFieldProto, nil
	case "method":
		return HTTPRequestFieldMethod, nil
	case "scheme":
		return HTTPRequestFieldScheme, nil
	case "host":
		return HTTPRequestFieldHost, nil
	case "path":
		return HTTPRequestFieldPath, nil
	case "query":
		return HTTPRequestFieldQuery, nil
	case "body":
		return HTTPRequestFieldBody, nil
	case "":
		return 0, fmt.Errorf("empty HTTP request field")
	default:
		return 0, fmt.Errorf("unknown HTTP request field: %s", s)
	}
}

// HTTPRequestOptions options to configure the HTTP fingerprint.
type HTTPRequestOptions struct {
	fields  uint64
	headers []string
	cookies []string
}

// HTTPRequestOption is a helper function to configure the HTTP fingerprint.
type HTTPRequestOption func(*HTTPRequestOptions)

// WithHTTPRequestFields sets the fields to be used to fingerprint the request.
func WithHTTPRequestFields(fields ...HTTPRequestField) HTTPRequestOption {
	return func(o *HTTPRequestOptions) {
		if len(fields) == 0 {
			return
		}
		o.fields = 0
		for _, field := range fields {
			o.fields |= uint64(field)
		}
	}
}

// WithHTTPRequestHeaders sets the headers to be used to fingerprint the request.
func WithHTTPRequestHeaders(headers ...string) HTTPRequestOption {
	return func(o *HTTPRequestOptions) {
		o.headers = headers
	}
}

// WithHTTPRequestCookies sets the cookies to be used to fingerprint the request.
func WithHTTPRequestCookies(cookies ...string) HTTPRequestOption {
	return func(o *HTTPRequestOptions) {
		o.cookies = cookies
	}
}

// HTTPRequest is a fingerprinter for web requests.
type HTTPRequest struct {
	*http.Request

	fields  uint64
	headers []string
	cookies []string
}

// NewHTTPRequest creates a new HTTPRequest fingerprinter.
func NewHTTPRequest(r *http.Request, options ...HTTPRequestOption) HTTPRequest {
	o := HTTPRequestOptions{
		fields: uint64(
			HTTPRequestFieldMethod |
				HTTPRequestFieldScheme |
				HTTPRequestFieldHost |
				HTTPRequestFieldPath |
				HTTPRequestFieldQuery,
		),
	}

	for _, opt := range options {
		opt(&o)
	}

	return HTTPRequest{
		Request: r,
		fields:  o.fields,
		headers: o.headers,
		cookies: o.cookies,
	}
}

// Fingerprint returns the fingerprint for the request.
func (r HTTPRequest) Fingerprint() anicetus.Fingerprint {
	var err error

	hash := sha256.New()
	if r.fields&uint64(HTTPRequestFieldProto) != 0 {
		if _, err = hash.Write([]byte(r.Proto)); err != nil {
			return ""
		}
	}
	if r.fields&uint64(HTTPRequestFieldMethod) != 0 {
		if _, err = hash.Write([]byte(r.Method)); err != nil {
			return ""
		}
	}
	if r.fields&uint64(HTTPRequestFieldScheme) != 0 {
		if _, err = hash.Write([]byte(r.URL.Scheme)); err != nil {
			return ""
		}
	}
	if r.fields&uint64(HTTPRequestFieldHost) != 0 {
		if r.Host != "" {
			if _, err = hash.Write([]byte(r.Host)); err != nil {
				return ""
			}
		} else {
			if _, err = hash.Write([]byte(r.URL.Host)); err != nil {
				return ""
			}
		}
	}
	if r.fields&uint64(HTTPRequestFieldPath) != 0 {
		if _, err = hash.Write([]byte(r.URL.Path)); err != nil {
			return ""
		}
	}
	if r.fields&uint64(HTTPRequestFieldQuery) != 0 {
		if _, err = hash.Write([]byte(r.URL.RawQuery)); err != nil {
			return ""
		}
	}
	if r.fields&uint64(HTTPRequestFieldBody) != 0 {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return ""
		}
		if _, err = hash.Write(body); err != nil {
			return ""
		}
		r.Body = io.NopCloser(bytes.NewBuffer(body))
	}
	for _, header := range r.headers {
		if _, err = hash.Write([]byte(r.Header.Get(header))); err != nil {
			return ""
		}
	}
	for _, cookie := range r.cookies {
		c, err := r.Cookie(cookie)
		if err != nil {
			return ""
		}
		if _, err = hash.Write([]byte(c.String())); err != nil {
			return ""
		}
	}
	return anicetus.Fingerprint(hex.EncodeToString(hash.Sum(nil)))
}
