package cmd

import "testing"

func TestTargetCommandMetadata(t *testing.T) {
	c := &TargetCommand{Manager: &Manager{}}

	if c.Name() != "target" {
		t.Fatalf("expected name %q, got %q", "target", c.Name())
	}
	if c.Description() == "" {
		t.Fatal("expected a non-empty description")
	}
}

func TestTargetCommandRegistered(t *testing.T) {
	cm, err := NewManager("example.com", nil)
	if err != nil {
		t.Fatalf("expected command manager, got error: %v", err)
	}

	got, err := cm.Get("target")
	if err != nil {
		t.Fatalf("expected target command to be registered, got error: %v", err)
	}
	if _, ok := got.(*TargetCommand); !ok {
		t.Fatalf("expected *TargetCommand, got %T", got)
	}
}

func TestNewManagerStoresTarget(t *testing.T) {
	cm, err := NewManager("example.com", nil)
	if err != nil {
		t.Fatalf("expected command manager, got error: %v", err)
	}

	if cm.Target != "example.com" {
		t.Fatalf("expected target %q, got %q", "example.com", cm.Target)
	}
}

func TestSetTargetUpdatesManager(t *testing.T) {
	cm := &Manager{Commands: make(map[string]Command), Target: "old.com"}

	cm.SetTarget("new.com")

	if cm.Target != "new.com" {
		t.Fatalf("expected target %q, got %q", "new.com", cm.Target)
	}
}

func TestTargetCommandSetsTarget(t *testing.T) {
	cm := &Manager{Commands: make(map[string]Command), Target: "old.com"}
	c := &TargetCommand{Manager: cm}

	out := captureStdout(t, func() {
		if err := c.Exec([]string{"new.com"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if cm.Target != "new.com" {
		t.Fatalf("expected target %q, got %q", "new.com", cm.Target)
	}
	if out == "" {
		t.Fatal("expected confirmation output, got none")
	}
}

func TestTargetCommandShowsCurrentTarget(t *testing.T) {
	cm := &Manager{Commands: make(map[string]Command), Target: "example.com"}
	c := &TargetCommand{Manager: cm}

	out := captureStdout(t, func() {
		if err := c.Exec(nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if cm.Target != "example.com" {
		t.Fatalf("expected target to be unchanged, got %q", cm.Target)
	}
	if out == "" {
		t.Fatal("expected target output, got none")
	}
}

func TestTargetCommandUsageErrors(t *testing.T) {
	cm := &Manager{Commands: make(map[string]Command), Target: "example.com"}
	c := &TargetCommand{Manager: cm}

	if err := c.Exec([]string{""}); err == nil {
		t.Fatal("expected error for empty target, got nil")
	}
	if err := c.Exec([]string{"a", "b"}); err == nil {
		t.Fatal("expected error for too many args, got nil")
	}
	if cm.Target != "example.com" {
		t.Fatalf("expected target to be unchanged after errors, got %q", cm.Target)
	}
}

func TestSetTargetIsLiveForCommands(t *testing.T) {
	cm := &Manager{Commands: make(map[string]Command), Target: "old.com"}
	ext := &ExternalCommand{Manager: cm, ModuleName: "demo", Module: "demo"}

	cm.SetTarget("new.com")

	if ext.Manager.Target != "new.com" {
		t.Fatalf("expected external command to see live target %q, got %q", "new.com", ext.Manager.Target)
	}
}
