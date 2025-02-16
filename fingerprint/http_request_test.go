package fingerprint_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/rafaeljusto/anicetus/v2"
	"github.com/rafaeljusto/anicetus/v2/fingerprint"
)

func TestHTTPRequest_Fingerprint(t *testing.T) {
	request := &http.Request{
		Proto:  "HTTP/1.1",
		Method: http.MethodGet,
		Host:   "example.com",
		URL: &url.URL{
			Scheme:   "http",
			Host:     "example2.com",
			Path:     "/",
			RawQuery: "key=value",
		},
		Header: http.Header{
			"User-Agent": []string{"Mozilla/5.0"},
			"Cookies":    []string{"cookiekey=cookievalue"},
		},
		Body: io.NopCloser(bytes.NewBufferString("body")),
	}

	tests := []struct {
		name    string
		options []fingerprint.HTTPRequestOption
		want    anicetus.Fingerprint
	}{{
		name: "it should match the fingerprint only for proto",
		options: []fingerprint.HTTPRequestOption{
			fingerprint.WithHTTPRequestFields(fingerprint.HTTPRequestFieldProto),
		},
		want: func() anicetus.Fingerprint {
			hash := sha256.New()
			_, _ = hash.Write([]byte(request.Proto))
			return anicetus.Fingerprint(hex.EncodeToString(hash.Sum(nil)))
		}(),
	}, {
		name: "it should match the fingerprint only for method",
		options: []fingerprint.HTTPRequestOption{
			fingerprint.WithHTTPRequestFields(fingerprint.HTTPRequestFieldMethod),
		},
		want: func() anicetus.Fingerprint {
			hash := sha256.New()
			_, _ = hash.Write([]byte(request.Method))
			return anicetus.Fingerprint(hex.EncodeToString(hash.Sum(nil)))
		}(),
	}, {
		name: "it should match the fingerprint only for scheme",
		options: []fingerprint.HTTPRequestOption{
			fingerprint.WithHTTPRequestFields(fingerprint.HTTPRequestFieldScheme),
		},
		want: func() anicetus.Fingerprint {
			hash := sha256.New()
			_, _ = hash.Write([]byte(request.URL.Scheme))
			return anicetus.Fingerprint(hex.EncodeToString(hash.Sum(nil)))
		}(),
	}, {
		name: "it should match the fingerprint only for host header",
		options: []fingerprint.HTTPRequestOption{
			fingerprint.WithHTTPRequestFields(fingerprint.HTTPRequestFieldHost),
		},
		want: func() anicetus.Fingerprint {
			hash := sha256.New()
			_, _ = hash.Write([]byte(request.Host))
			return anicetus.Fingerprint(hex.EncodeToString(hash.Sum(nil)))
		}(),
	}, {
		name: "it should match the fingerprint only for path",
		options: []fingerprint.HTTPRequestOption{
			fingerprint.WithHTTPRequestFields(fingerprint.HTTPRequestFieldPath),
		},
		want: func() anicetus.Fingerprint {
			hash := sha256.New()
			_, _ = hash.Write([]byte(request.URL.Path))
			return anicetus.Fingerprint(hex.EncodeToString(hash.Sum(nil)))
		}(),
	}, {
		name: "it should match the fingerprint only for query",
		options: []fingerprint.HTTPRequestOption{
			fingerprint.WithHTTPRequestFields(fingerprint.HTTPRequestFieldQuery),
		},
		want: func() anicetus.Fingerprint {
			hash := sha256.New()
			_, _ = hash.Write([]byte(request.URL.RawQuery))
			return anicetus.Fingerprint(hex.EncodeToString(hash.Sum(nil)))
		}(),
	}, {
		name: "it should match the fingerprint only for body",
		options: []fingerprint.HTTPRequestOption{
			fingerprint.WithHTTPRequestFields(fingerprint.HTTPRequestFieldBody),
		},
		want: func() anicetus.Fingerprint {
			body, _ := io.ReadAll(request.Body)
			request.Body = io.NopCloser(bytes.NewBuffer(body))

			hash := sha256.New()
			_, _ = hash.Write(body)
			return anicetus.Fingerprint(hex.EncodeToString(hash.Sum(nil)))
		}(),
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fingerprint.NewHTTPRequest(request, tt.options...).Fingerprint()
			if got != tt.want {
				t.Errorf("unexpected result: got %v, want %v", got, tt.want)
			}
		})
	}
}
