package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goscouter/internal/module"
)

func TestUninstallCommandMetadata(t *testing.T) {
	c := &UninstallCommand{}
	if c.Name() != "uninstall" {
		t.Fatalf("expected name %q, got %q", "uninstall", c.Name())
	}
	if c.Description() == "" {
		t.Fatal("expected a non-empty description")
	}
}

func TestUninstallExecUsage(t *testing.T) {
	c := &UninstallCommand{}

	if err := c.Exec(nil); err == nil {
		t.Fatal("expected error when no args are given")
	}
	if err := c.Exec([]string{""}); err == nil {
		t.Fatal("expected error for empty module name")
	}
	if err := c.Exec([]string{"a", "b"}); err == nil {
		t.Fatal("expected error for too many args")
	}
}

func TestUninstallCommandUnregisters(t *testing.T) {
	cm, err := NewManager("", nil)
	if err != nil {
		t.Fatalf("expected command manager, got error: %v", err)
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		t.Fatalf("resolving cache dir: %v", err)
	}
	dir := filepath.Join(cacheDir, "gs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating cache dir: %v", err)
	}

	binaryPath := filepath.Join(dir, "demo-uninstall"+execSuffix())
	if err := os.WriteFile(binaryPath, []byte("binary"), 0o755); err != nil {
		t.Fatalf("seeding module binary: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(binaryPath) })

	name := commandName(binaryPath)
	cm.Add(&ExternalCommand{ModuleName: name, Module: binaryPath})

	c := &UninstallCommand{Manager: cm}
	out := captureStdout(t, func() {
		if err := c.Exec([]string{name}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if _, err := os.Stat(binaryPath); !os.IsNotExist(err) {
		t.Fatalf("expected binary to be removed, stat err = %v", err)
	}
	if _, err := cm.Get(name); err == nil {
		t.Fatal("expected command to be unregistered")
	}
	if !strings.Contains(out, "no longer available") {
		t.Fatalf("expected uninstall output, got %q", out)
	}
}

func TestUninstallExecNotInstalled(t *testing.T) {
	cm, err := NewManager("", nil)
	if err != nil {
		t.Fatalf("expected command manager, got error: %v", err)
	}

	c := &UninstallCommand{Manager: cm}
	err = c.Exec([]string{"definitely-not-installed"})
	if err == nil {
		t.Fatal("expected error for module that is not installed, got nil")
	}
	if !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("expected not-installed error, got %v", err)
	}
}

func TestRemoveFromSuccess(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "demo")
	if err := os.WriteFile(binaryPath, []byte("binary"), 0o755); err != nil {
		t.Fatalf("seeding module binary: %v", err)
	}

	out := captureStdout(t, func() {
		got, err := module.RemoveFrom("demo", dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != binaryPath {
			t.Fatalf("expected removed path %q, got %q", binaryPath, got)
		}
	})

	if _, err := os.Stat(binaryPath); !os.IsNotExist(err) {
		t.Fatalf("expected binary to be removed, stat err = %v", err)
	}
	if !strings.Contains(out, "Uninstalled") {
		t.Fatalf("expected uninstall output, got %q", out)
	}
}

func TestRemoveFromNotInstalled(t *testing.T) {
	_, err := module.RemoveFrom("missing", t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing module, got nil")
	}
	if !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("expected not-installed error, got %v", err)
	}
}

func TestRemoveFromRejectsDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "demo"), 0o755); err != nil {
		t.Fatalf("creating dir: %v", err)
	}

	_, err := module.RemoveFrom("demo", dir)
	if err == nil {
		t.Fatal("expected error for directory, got nil")
	}
	if !strings.Contains(err.Error(), "not an installed module") {
		t.Fatalf("expected directory rejection error, got %v", err)
	}
}
