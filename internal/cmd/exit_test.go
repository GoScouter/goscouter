package cmd

import (
	"errors"
	"testing"
)

func TestExitCommandMetadata(t *testing.T) {
	c := &ExitCommand{}
	if c.Name() != "exit" {
		t.Fatalf("expected name %q, got %q", "exit", c.Name())
	}
	if c.Description() == "" {
		t.Fatal("expected a non-empty description")
	}
}

func TestExitCommandExec(t *testing.T) {
	c := &ExitCommand{}

	err := c.Exec(nil)
	if !errors.Is(err, ErrExit) {
		t.Fatalf("expected ErrExit, got %v", err)
	}
}

func TestExitCommandExecIgnoresArgs(t *testing.T) {
	c := &ExitCommand{}

	if err := c.Exec([]string{"some", "args"}); !errors.Is(err, ErrExit) {
		t.Fatalf("expected ErrExit regardless of args, got %v", err)
	}
}
