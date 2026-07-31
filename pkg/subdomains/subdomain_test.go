package subdomains

import (
	"strings"
	"testing"
	"time"
)

func TestSubdomainRender(t *testing.T) {
	seen := time.Date(2022, 12, 1, 15, 4, 5, 0, time.UTC)
	s := &Subdomain{Name: "api.example.com", LastSeen: seen}

	got := s.Render()
	want := "[+] Found: api.example.com (2022-12-01T15:04:05Z)"
	if got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestRenderAllOneHitPerLine(t *testing.T) {
	seen := time.Date(2022, 12, 1, 15, 4, 5, 0, time.UTC)
	out := RenderAll([]Subdomain{
		{Name: "a.example.com", LastSeen: seen},
		{Name: "b.example.com", LastSeen: seen},
	})

	lines := strings.Split(strings.TrimSpace(out), "\r\n")
	if len(lines) != 2 {
		t.Fatalf("RenderAll() produced %d lines, want 2\ngot:\n%s", len(lines), out)
	}
	for i, want := range []string{"a.example.com", "b.example.com"} {
		if !strings.HasPrefix(lines[i], "[+] Found: "+want) {
			t.Errorf("line %d = %q, want a [+] hit for %q", i, lines[i], want)
		}
	}
}

func TestRenderAllEmpty(t *testing.T) {
	out := RenderAll(nil)

	if !strings.HasPrefix(strings.TrimSpace(out), "[-] No subdomains found") {
		t.Errorf("RenderAll() with no results = %q, want a [-] line", out)
	}
}
