package utils

import (
	"strings"
	"testing"
)

func TestBanner(t *testing.T) {
	const (
		version   = "v1.2.3"
		buildTime = "2026-07-18"
	)

	out := Banner(version, buildTime)

	for _, want := range []string{version, buildTime, NAME} {
		if !strings.Contains(out, want) {
			t.Errorf("Banner output missing %q\ngot:\n%s", want, out)
		}
	}

	if !strings.Contains(out, "██████╗") {
		t.Errorf("Banner output missing ASCII art\ngot:\n%s", out)
	}

	if !strings.Contains(out, "\r\n") {
		t.Errorf("Banner output missing CRLF line endings\ngot:\n%s", out)
	}
}

func TestBannerVariesWithArgs(t *testing.T) {
	a := Banner("v1.0.0", "monday")
	b := Banner("v2.0.0", "tuesday")

	if a == b {
		t.Errorf("Banner produced identical output for different args:\n%s", a)
	}
}

func TestBannerEmptyArgs(t *testing.T) {
	out := Banner("", "")

	if !strings.Contains(out, NAME) {
		t.Errorf("Banner output missing %q for empty args\ngot:\n%s", NAME, out)
	}
}
