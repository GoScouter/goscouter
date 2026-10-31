package subdomain

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestFindAllCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	subs, err := FindAll(ctx, "example.com")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v (%d subs)", err, len(subs))
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("expected a cancelled search to return at once, took %v", elapsed)
	}
}
