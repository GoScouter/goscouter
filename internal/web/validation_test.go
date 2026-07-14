package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"goscouter/internal/logger"
)

func TestMain(m *testing.M) {
	logger.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	os.Exit(m.Run())
}

func TestCheckSiteStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		wantCode int
	}{
		{"ok", http.StatusOK, http.StatusOK},
		{"not found", http.StatusNotFound, http.StatusNotFound},
		{"server error", http.StatusInternalServerError, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()

			code, err := CheckSiteStatus(srv.URL)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if code != tt.wantCode {
				t.Fatalf("expected code %d, got %d", tt.wantCode, code)
			}
		})
	}
}

func TestCheckSiteStatusBadURL(t *testing.T) {
	if _, err := CheckSiteStatus("://not-a-url"); err == nil {
		t.Fatal("expected error for malformed URL, got nil")
	}
}

func TestCheckSiteStatusUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	if _, err := CheckSiteStatus(url); err == nil {
		t.Fatal("expected error for unreachable host, got nil")
	}
}
