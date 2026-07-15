package style

import (
	"strings"
	"testing"
)

func withEnabled(t *testing.T, want bool, fn func()) {
	t.Helper()
	prev := enabled
	enabled = want
	defer func() { enabled = prev }()
	fn()
}

func TestDisabledIsPlain(t *testing.T) {
	withEnabled(t, false, func() {
		got := Red("boom")
		if got != "boom" {
			t.Fatalf("Red with styling off = %q, want plain %q", got, "boom")
		}
		if strings.Contains(Prompt(), "\033") {
			t.Fatalf("Prompt emitted escape codes while styling disabled: %q", Prompt())
		}
	})
}

func TestEnabledWraps(t *testing.T) {
	withEnabled(t, true, func() {
		got := Red("boom")
		if !strings.HasPrefix(got, codeRed) || !strings.HasSuffix(got, reset) {
			t.Fatalf("Red = %q, want wrapped in color + reset", got)
		}
	})
}

func TestSemanticPrefixes(t *testing.T) {
	withEnabled(t, false, func() {
		cases := map[string]string{
			"✗ ": Error("x"),
			"✓ ": Success("x"),
			"» ": Info("x"),
		}
		for prefix, out := range cases {
			if !strings.HasPrefix(out, prefix) {
				t.Errorf("output %q missing prefix %q", out, prefix)
			}
		}
	})
}
