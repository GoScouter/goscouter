package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"goscouter/internal/module"
)

func checksumOf(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func serveBinary(t *testing.T, payload []byte, status int) (module.Manifest, func()) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		_, _ = w.Write(payload)
	}))

	manifest := module.Manifest{
		Name:    "demo",
		Version: "1.0.0",
		Platforms: map[string]module.Platform{
			runtime.GOOS: {
				Checksum: checksumOf(payload),
				Binary:   srv.URL + "/demo",
			},
		},
	}

	return manifest, srv.Close
}

func TestInstallCommandMetadata(t *testing.T) {
	c := &InstallCommand{}
	if c.Name() != "install" {
		t.Fatalf("expected name %q, got %q", "install", c.Name())
	}
	if c.Description() == "" {
		t.Fatal("expected a non-empty description")
	}
}

func TestInstallExecUsage(t *testing.T) {
	c := &InstallCommand{}

	if err := c.Exec(nil); err == nil {
		t.Fatal("expected error when no args are given")
	}
	if err := c.Exec([]string{""}); err == nil {
		t.Fatal("expected error for empty module ref")
	}
	if err := c.Exec([]string{"a", "b"}); err == nil {
		t.Fatal("expected error for too many args")
	}
}

func TestDownloadToSuccess(t *testing.T) {
	payload := []byte("#!/bin/sh\necho hello\n")
	manifest, closeSrv := serveBinary(t, payload, http.StatusOK)
	defer closeSrv()

	dir := t.TempDir()
	out := captureStdout(t, func() {
		if err := downloadTo(manifest, dir); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	binaryPath := filepath.Join(dir, "demo")
	info, err := os.Stat(binaryPath)
	if err != nil {
		t.Fatalf("expected installed binary, got error: %v", err)
	}

	got, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("reading binary: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("binary content mismatch, got %q", string(got))
	}

	if runtime.GOOS != "windows" {
		if info.Mode().Perm()&0o111 == 0 {
			t.Fatalf("expected binary to be executable, got mode %v", info.Mode().Perm())
		}
	}

	if !strings.Contains(out, "Installed") {
		t.Fatalf("expected install output, got %q", out)
	}
}

func TestDownloadToCreatesMissingDir(t *testing.T) {
	payload := []byte("binary")
	manifest, closeSrv := serveBinary(t, payload, http.StatusOK)
	defer closeSrv()

	dir := filepath.Join(t.TempDir(), "nested", "gs")
	_ = captureStdout(t, func() {
		if err := downloadTo(manifest, dir); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if _, err := os.Stat(filepath.Join(dir, "demo")); err != nil {
		t.Fatalf("expected binary in created dir: %v", err)
	}
}

func TestDownloadToChecksumMismatch(t *testing.T) {
	payload := []byte("real content")
	manifest, closeSrv := serveBinary(t, payload, http.StatusOK)
	defer closeSrv()

	p := manifest.Platforms[runtime.GOOS]
	p.Checksum = "deadbeef"
	manifest.Platforms[runtime.GOOS] = p

	dir := t.TempDir()
	err := downloadTo(manifest, dir)
	if err == nil {
		t.Fatal("expected checksum mismatch error, got nil")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch error, got %v", err)
	}

	if _, statErr := os.Stat(filepath.Join(dir, "demo")); !os.IsNotExist(statErr) {
		t.Fatal("expected corrupt binary to be removed")
	}
}

func TestDownloadToAlreadyInstalled(t *testing.T) {
	payload := []byte("binary")
	manifest, closeSrv := serveBinary(t, payload, http.StatusOK)
	defer closeSrv()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "demo"), []byte("old"), 0o755); err != nil {
		t.Fatalf("seeding existing module: %v", err)
	}

	err := downloadTo(manifest, dir)
	if err == nil {
		t.Fatal("expected error for already-installed module, got nil")
	}
	if !strings.Contains(err.Error(), "already installed") {
		t.Fatalf("expected already-installed error, got %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "demo"))
	if err != nil {
		t.Fatalf("reading existing binary: %v", err)
	}
	if string(got) != "old" {
		t.Fatalf("expected existing binary to be preserved, got %q", string(got))
	}
}

func TestDownloadToUnsupportedPlatform(t *testing.T) {
	manifest := module.Manifest{
		Name:      "demo",
		Version:   "1.0.0",
		Platforms: map[string]module.Platform{"nonexistent-os": {}},
	}

	err := downloadTo(manifest, t.TempDir())
	if err == nil {
		t.Fatal("expected error for unsupported platform, got nil")
	}
	if !strings.Contains(err.Error(), "platform") {
		t.Fatalf("expected platform error, got %v", err)
	}
}

func TestDownloadToNotFound(t *testing.T) {
	manifest, closeSrv := serveBinary(t, nil, http.StatusNotFound)
	defer closeSrv()

	err := downloadTo(manifest, t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing binary, got nil")
	}
	if !strings.Contains(err.Error(), "HTTP 404") {
		t.Fatalf("expected HTTP 404 error, got %v", err)
	}
}
