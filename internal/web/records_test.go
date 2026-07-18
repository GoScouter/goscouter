package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchHTTPRecords(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "value")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rec, err := FetchHTTPRecords(srv.URL, "HTTP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Scheme != "HTTP" {
		t.Errorf("Scheme = %q, want %q", rec.Scheme, "HTTP")
	}
	if rec.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", rec.StatusCode, http.StatusOK)
	}
	if rec.RequestURL != srv.URL {
		t.Errorf("RequestURL = %q, want %q", rec.RequestURL, srv.URL)
	}
	if got := rec.Headers.Get("X-Test"); got != "value" {
		t.Errorf("Headers[X-Test] = %q, want %q", got, "value")
	}
}

func TestFetchHTTPRecordsCapturesRedirect(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer final.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))
	defer redirector.Close()

	rec, err := FetchHTTPRecords(redirector.URL, "HTTP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.RequestURL != redirector.URL {
		t.Errorf("RequestURL = %q, want %q", rec.RequestURL, redirector.URL)
	}
	if rec.FinalURL != final.URL {
		t.Errorf("FinalURL = %q, want %q (should follow redirect)", rec.FinalURL, final.URL)
	}
}

func TestFetchHTTPRecordsBadURL(t *testing.T) {
	if _, err := FetchHTTPRecords("://nope", "HTTP"); err == nil {
		t.Fatal("expected an error for a malformed URL, got nil")
	}
}
