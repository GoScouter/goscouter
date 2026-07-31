package cmd

import (
	"runtime"
	"strings"
	"testing"

	"goscouter/internal"
)

func TestInfoCommandMetadata(t *testing.T) {
	c := &InfoCommand{}
	if c.Name() != "info" {
		t.Fatalf("expected name %q, got %q", "info", c.Name())
	}
	if c.Description() == "" {
		t.Fatal("expected a non-empty description")
	}
}

func TestInfoCommandExec(t *testing.T) {
	internal.Version = "1.2.3"

	c := &InfoCommand{}

	out := captureStdout(t, func() {
		if err := c.Exec(nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	wants := []string{
		"GoScouter",
		"github.com/GoScouter/goscouter",
		"https://goscouter.github.io/",
		"1.2.3",
		strings.TrimPrefix(runtime.Version(), "go"),
		runtime.GOOS + " / " + runtime.GOARCH,
		"GPL-3.0",
		"Purpose",
	}
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Fatalf("expected info output to contain %q, got:\n%s", want, out)
		}
	}
}
