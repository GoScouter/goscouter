package subdomains

import (
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
