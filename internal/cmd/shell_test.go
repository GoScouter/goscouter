package cmd

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestRunShellCancelled(t *testing.T) {
	line := "sleep 30 | cat"
	if runtime.GOOS == "windows" {
		line = "ping -n 30 127.0.0.1 | find \"x\""
	}

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(300*time.Millisecond, cancel)

	start := time.Now()
	err := RunShell(ctx, line)
	if err == nil || err.Error() != "interrupted" {
		t.Fatalf("expected the command to be interrupted, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("expected cancelling to stop the command, took %v", elapsed)
	}
}
