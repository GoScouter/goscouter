package subdomain

import (
	"testing"
	"time"
)

func TestNormalize(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Example.com", "example.com"},
		{"  api.example.com  ", "api.example.com"},
		{"*.example.com", "example.com"},
		{"*.API.Example.COM", "api.example.com"},
		{"", ""},
		{"   ", ""},
	}
	for _, c := range cases {
		if got := normalize(c.in); got != c.want {
			t.Errorf("normalize(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestKeepLatestKeepsNewer(t *testing.T) {
	latest := make(map[string]time.Time)
	older := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	keepLatest(latest, "api.example.com", older)
	keepLatest(latest, "api.example.com", newer)

	if got := latest["api.example.com"]; !got.Equal(newer) {
		t.Errorf("keepLatest kept %v, want the newer %v", got, newer)
	}
}

func TestKeepLatestIgnoresOlder(t *testing.T) {
	latest := make(map[string]time.Time)
	older := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	keepLatest(latest, "api.example.com", newer)
	keepLatest(latest, "api.example.com", older)

	if got := latest["api.example.com"]; !got.Equal(newer) {
		t.Errorf("keepLatest kept %v, want the newer %v", got, newer)
	}
}

func TestKeepLatestNormalizesKey(t *testing.T) {
	latest := make(map[string]time.Time)
	t0 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	keepLatest(latest, "*.API.Example.com", t0)

	if _, ok := latest["api.example.com"]; !ok {
		t.Errorf("keepLatest should store the normalized key, got keys: %v", latest)
	}
}

func TestKeepLatestSkipsEmptyName(t *testing.T) {
	latest := make(map[string]time.Time)
	keepLatest(latest, "  ", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC))
	if len(latest) != 0 {
		t.Errorf("keepLatest should skip empty names, got: %v", latest)
	}
}

func TestFlattenSortsByName(t *testing.T) {
	t0 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	latest := map[string]time.Time{
		"c.example.com": t0,
		"a.example.com": t0,
		"b.example.com": t0,
	}

	out := flatten(latest)

	if len(out) != 3 {
		t.Fatalf("flatten returned %d entries, want 3", len(out))
	}
	want := []string{"a.example.com", "b.example.com", "c.example.com"}
	for i, w := range want {
		if out[i].Name != w {
			t.Errorf("flatten[%d].Name = %q, want %q", i, out[i].Name, w)
		}
	}
}

func TestFlattenEmpty(t *testing.T) {
	out := flatten(map[string]time.Time{})
	if len(out) != 0 {
		t.Errorf("flatten of empty map returned %d entries, want 0", len(out))
	}
}
