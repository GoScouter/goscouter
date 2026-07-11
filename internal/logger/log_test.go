package logger

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogPath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	got, err := LogPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if filepath.Base(got) != "goscouter.log" {
		t.Fatalf("expected file goscouter.log, got %q", got)
	}
	if !strings.HasSuffix(filepath.Dir(got), "goscouter") {
		t.Fatalf("expected path under goscouter dir, got %q", got)
	}

	if info, err := os.Stat(filepath.Dir(got)); err != nil || !info.IsDir() {
		t.Fatalf("expected log directory to exist, err=%v", err)
	}
}

func TestSetupLogger(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := SetupLogger(LoggerConfig{Console: false, Level: slog.LevelInfo}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if Log == nil {
		t.Fatal("expected Log to be initialized")
	}

	Log.Info("hello from test", "key", "value")

	data, err := os.ReadFile(filepath.Join(home, "goscouter", "goscouter.log"))
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}
	if !strings.Contains(string(data), "hello from test") {
		t.Fatalf("expected log message in file, got %q", string(data))
	}
}
