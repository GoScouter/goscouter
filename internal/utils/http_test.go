package utils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"goscouter/internal"
)

func TestGetSendsVersionedUserAgent(t *testing.T) {
	var ua string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua = r.Header.Get("User-Agent")
	}))
	defer srv.Close()

	tests := []struct {
		version string
		want    string
	}{
		{"1.2.3", "goscouter/1.2.3"},
		{"", "goscouter/dev"},
	}

	for _, tt := range tests {
		internal.Version = tt.version
		resp, err := Get(context.Background(), srv.URL)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		resp.Body.Close()

		if ua != tt.want {
			t.Errorf("version %q: User-Agent = %q, want %q", tt.version, ua, tt.want)
		}
	}
}

func TestHTTPClientOverridesUserAgent(t *testing.T) {
	internal.Version = "1.2.3"

	var ua string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua = r.Header.Get("User-Agent")
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", "other")

	resp, err := HTTPClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	resp.Body.Close()

	if ua != "goscouter/1.2.3" {
		t.Errorf("User-Agent = %q, want %q", ua, "goscouter/1.2.3")
	}
	if req.Header.Get("User-Agent") != "other" {
		t.Error("caller's request was modified")
	}
}
