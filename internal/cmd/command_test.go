package cmd

import (
	"context"
	"errors"
	"slices"
	"testing"
)

type stubCommand struct {
	name        string
	description string
	execErr     error
}

func (c *stubCommand) Name() string                                  { return c.name }
func (c *stubCommand) Description() string                           { return c.description }
func (c *stubCommand) Exec(ctx context.Context, args []string) error { return c.execErr }

func TestNewCommandManagerRegistersExit(t *testing.T) {
	cm, err := NewManager("", nil)
	if err != nil {
		t.Fatalf("expected command manager, got error: %v", err)
	}

	got, err := cm.Get("exit")
	if err != nil {
		t.Fatalf("expected exit command to be registered, got error: %v", err)
	}
	if _, ok := got.(*ExitCommand); !ok {
		t.Fatalf("expected *ExitCommand, got %T", got)
	}
}

func TestGetCommandUnknown(t *testing.T) {
	cm, err := NewManager("", nil)
	if err != nil {
		t.Fatalf("expected command manager, got error: %v", err)
	}

	if _, err := cm.Get("nope"); err == nil {
		t.Fatal("expected error for unknown command, got nil")
	}
}

func TestAddAndGetCommand(t *testing.T) {
	cm, err := NewManager("", nil)
	if err != nil {
		t.Fatalf("expected command manager, got error: %v", err)
	}

	cmd := &stubCommand{name: "hello"}
	cm.addCommand(cmd)

	got, err := cm.Get("hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != cmd {
		t.Fatalf("expected the added command back, got %#v", got)
	}
}

func TestRemoveCommand(t *testing.T) {
	cm, err := NewManager("", nil)
	if err != nil {
		t.Fatalf("expected command manager, got error: %v", err)
	}

	cm.addCommand(&stubCommand{name: "hello"})
	cm.removeCommand("hello")

	if _, err := cm.Get("hello"); err == nil {
		t.Fatal("expected error after removing command, got nil")
	}
}

func TestExitCommand(t *testing.T) {
	cmd := &ExitCommand{}

	if cmd.Name() != "exit" {
		t.Fatalf("expected name %q, got %q", "exit", cmd.Name())
	}
	if err := cmd.Exec(context.Background(), nil); !errors.Is(err, ErrExit) {
		t.Fatalf("expected ErrExit, got %v", err)
	}
}

func TestCompleteMatchesFullAndShortNames(t *testing.T) {
	cm := &CommandManager{Commands: make(map[string]Command)}
	for _, name := range []string{"help", "info", "internal:dns", "internal:subdomains", "author:subdomains", "author:whois"} {
		cm.addCommand(&stubCommand{name: name})
	}

	cases := []struct {
		word string
		want []string
	}{
		{"he", []string{"help"}},
		{"in", []string{"info", "internal:dns", "internal:subdomains"}},
		{"dn", []string{"dns"}},
		{"who", []string{"whois"}},
		{"subdo", []string{"author:subdomains", "internal:subdomains"}},
		{"author:", []string{"author:subdomains", "author:whois"}},
		{"zzz", nil},
	}
	for _, c := range cases {
		if got := cm.Complete(c.word); !slices.Equal(got, c.want) {
			t.Errorf("Complete(%q) = %v, want %v", c.word, got, c.want)
		}
	}
}

func TestGetPrefersExactShortName(t *testing.T) {
	cm := &CommandManager{Commands: make(map[string]Command)}
	for _, name := range []string{"author:allsubdomains", "internal:subdomains"} {
		cm.addCommand(&stubCommand{name: name})
	}

	got, err := cm.Get("subdomains")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name() != "internal:subdomains" {
		t.Fatalf("expected internal:subdomains, got %s", got.Name())
	}
}

func TestGetMatchesOnlyWholeNames(t *testing.T) {
	cm := &CommandManager{Commands: make(map[string]Command)}
	for _, name := range []string{"help", "internal:dns", "internal:subdomains", "author:subdomains"} {
		cm.addCommand(&stubCommand{name: name})
	}

	cases := []struct {
		input string
		want  string // empty means an error
	}{
		{"help", "help"},
		{"internal:dns", "internal:dns"},
		{"dns", "internal:dns"},
		{"s", ""},
		{"p", ""},
		{"ns", ""},
		{"subdomains", ""},
		{"author:subdomains", "author:subdomains"},
	}
	for _, c := range cases {
		got, err := cm.Get(c.input)
		switch {
		case c.want == "" && err == nil:
			t.Errorf("Get(%q) = %s, want an error", c.input, got.Name())
		case c.want != "" && err != nil:
			t.Errorf("Get(%q) failed: %v", c.input, err)
		case c.want != "" && got.Name() != c.want:
			t.Errorf("Get(%q) = %s, want %s", c.input, got.Name(), c.want)
		}
	}
}
