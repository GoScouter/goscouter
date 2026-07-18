package cmd

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"

	"goscouter/internal/logger"
)

func TestMain(m *testing.M) {
	logger.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	os.Exit(m.Run())
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		out, _ := io.ReadAll(r)
		done <- string(out)
	}()

	fn()

	w.Close()
	os.Stdout = orig
	return <-done
}

func TestClearCommandMetadata(t *testing.T) {
	c := &ClearCommand{}
	if c.Name() != "clear" {
		t.Fatalf("expected name %q, got %q", "clear", c.Name())
	}
	if c.Description() == "" {
		t.Fatal("expected a non-empty description")
	}
}

func TestClearCommandExec(t *testing.T) {
	c := &ClearCommand{}

	out := captureStdout(t, func() {
		if err := c.Exec(nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, clear) {
		t.Fatalf("expected output to contain the clear escape sequence, got %q", out)
	}
}
