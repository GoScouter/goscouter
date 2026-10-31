package module

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/GoScouter/sdk"
)

type slowModule struct{}

func (slowModule) Info() sdk.ModuleInfo          { return sdk.ModuleInfo{Name: "slow", Author: "test"} }
func (slowModule) Render(json.RawMessage) string { return "" }
func (slowModule) Scout(string, []string) (json.RawMessage, error) {
	time.Sleep(time.Minute)
	return nil, nil
}

type contextModule struct{ slowModule }

func (contextModule) ScoutContext(ctx context.Context, _ string, _ []string) (json.RawMessage, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestScoutCancelled(t *testing.T) {
	for name, m := range map[string]sdk.Module{"plain": slowModule{}, "context aware": contextModule{}} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			time.AfterFunc(20*time.Millisecond, cancel)

			start := time.Now()
			_, err := scout(ctx, m, "example.com", nil)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("expected context.Canceled, got %v", err)
			}
			if elapsed := time.Since(start); elapsed > time.Second {
				t.Fatalf("expected cancelling to return right away, took %v", elapsed)
			}
		})
	}
}

func TestScoutReturnsResult(t *testing.T) {
	data, err := scout(context.Background(), &HttpModuleStub{}, "example.com", nil)
	if err != nil || string(data) != `"ok"` {
		t.Fatalf("expected the module's result, got %s, %v", data, err)
	}
}

type HttpModuleStub struct{ slowModule }

func (*HttpModuleStub) Scout(string, []string) (json.RawMessage, error) {
	return json.RawMessage(`"ok"`), nil
}
