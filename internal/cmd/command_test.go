package cmd

import (
	"errors"
	"testing"
)

type stubCommand struct {
	name    string
	execErr error
}

func (c *stubCommand) Name() string             { return c.name }
func (c *stubCommand) Exec(args []string) error { return c.execErr }

func TestNewCommandManagerRegistersExit(t *testing.T) {
	cm := NewCommandManager()

	got, err := cm.GetCommand("exit")
	if err != nil {
		t.Fatalf("expected exit command to be registered, got error: %v", err)
	}
	if _, ok := got.(*ExitCommand); !ok {
		t.Fatalf("expected *ExitCommand, got %T", got)
	}
}

func TestGetCommandUnknown(t *testing.T) {
	cm := NewCommandManager()

	if _, err := cm.GetCommand("nope"); err == nil {
		t.Fatal("expected error for unknown command, got nil")
	}
}

func TestAddAndGetCommand(t *testing.T) {
	cm := NewCommandManager()
	cmd := &stubCommand{name: "hello"}
	cm.AddCommand(cmd)

	got, err := cm.GetCommand("hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != cmd {
		t.Fatalf("expected the added command back, got %#v", got)
	}
}

func TestRemoveCommand(t *testing.T) {
	cm := NewCommandManager()
	cm.AddCommand(&stubCommand{name: "hello"})
	cm.RemoveCommand("hello")

	if _, err := cm.GetCommand("hello"); err == nil {
		t.Fatal("expected error after removing command, got nil")
	}
}

func TestExitCommand(t *testing.T) {
	cmd := &ExitCommand{}

	if cmd.Name() != "exit" {
		t.Fatalf("expected name %q, got %q", "exit", cmd.Name())
	}
	if err := cmd.Exec(nil); !errors.Is(err, ErrExit) {
		t.Fatalf("expected ErrExit, got %v", err)
	}
}
