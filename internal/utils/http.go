package utils

import (
	"context"
	"net/http"

	"goscouter/internal"
)

var fixedHeaders = map[string]string{}

// internal.Version is set at startup, so it's read per request.
func userAgent() string {
	version := internal.Version
	if version == "" {
		version = "dev"
	}
	return "goscouter/" + version
}

type headerTransport struct {
	base http.RoundTripper
}

func (t headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	for k, v := range fixedHeaders {
		req.Header.Set(k, v)
	}
	req.Header.Set("User-Agent", userAgent())
	return t.base.RoundTrip(req)
}

var HTTPClient = &http.Client{
	Transport: headerTransport{base: http.DefaultTransport},
}

func Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return HTTPClient.Do(req)
}
