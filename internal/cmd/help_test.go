package cmd

import (
	"strings"
	"testing"
)

func TestHelpCommandMetadata(t *testing.T) {
	c := &HelpCommand{}
	if c.Name() != "help" {
		t.Fatalf("expected name %q, got %q", "help", c.Name())
	}
	if c.Description() == "" {
		t.Fatal("expected a non-empty description")
	}
}

func TestHelpCommandExecListsCommands(t *testing.T) {
	c := &HelpCommand{
		Commands: []Command{
			&stubCommand{name: "alpha", description: "first"},
			&stubCommand{name: "beta", description: "second"},
		},
	}

	out := captureStdout(t, func() {
		if err := c.Exec(nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, want := range []string{"help", "alpha", "first", "beta", "second", "--target"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected help output to contain %q, got:\n%s", want, out)
		}
	}
}
