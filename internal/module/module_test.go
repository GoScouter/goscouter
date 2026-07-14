package module

import (
	"testing"

	"github.com/GoScouter/sdk"
)

func TestManagerRegistersModules(t *testing.T) {
	m := NewManager()

	for _, name := range []string{"dns", "subdomains", "http"} {
		mod, err := m.Get(name)
		if err != nil {
			t.Errorf("Get(%q): unexpected error: %v", name, err)
			continue
		}
		if mod.Name() != name {
			t.Errorf("Get(%q).Name() = %q, want %q", name, mod.Name(), name)
		}
	}
}

func TestManagerGetUnknown(t *testing.T) {
	m := NewManager()
	if _, err := m.Get("does-not-exist"); err == nil {
		t.Fatal("Get of an unknown module should return an error, got nil")
	}
}

func TestManagerGetAll(t *testing.T) {
	m := NewManager()
	all := m.GetAll()
	if len(all) != 3 {
		t.Fatalf("GetAll() returned %d modules, want 3", len(all))
	}
}

func TestManagerAddOverwrites(t *testing.T) {
	m := NewManager()
	before := len(m.GetAll())
	m.Add(&HttpModule{})
	if after := len(m.GetAll()); after != before {
		t.Errorf("Add of an existing module changed count from %d to %d", before, after)
	}
}

func TestModuleMetadata(t *testing.T) {
	cases := []struct {
		mod  sdk.Module
		name string
	}{
		{&RecordsModule{}, "dns"},
		{&SubdomainsModule{}, "subdomains"},
		{&HttpModule{}, "http"},
	}
	for _, c := range cases {
		if c.mod.Name() != c.name {
			t.Errorf("Name() = %q, want %q", c.mod.Name(), c.name)
		}
		if c.mod.Description() == "" {
			t.Errorf("%s: Description() should not be empty", c.name)
		}
		if c.mod.Version() == "" {
			t.Errorf("%s: Version() should not be empty", c.name)
		}
	}
}

func TestForceScheme(t *testing.T) {
	cases := []struct {
		target string
		scheme string
		want   string
	}{
		{"example.com", "https", "https://example.com"},
		{"example.com", "http", "http://example.com"},
		{"http://example.com", "https", "https://example.com"},
		{"https://example.com", "http", "http://example.com"},
		{"http://example.com/path", "https", "https://example.com/path"},
		{"ftp://example.com", "http", "http://example.com"},
		{"example.com:8080", "https", "https://example.com:8080"},
	}
	for _, c := range cases {
		if got := forceScheme(c.target, c.scheme); got != c.want {
			t.Errorf("forceScheme(%q, %q) = %q, want %q", c.target, c.scheme, got, c.want)
		}
	}
}

func TestSubdomainResultsRender(t *testing.T) {
	var r subdomainResults
	if got := r.Render(); got != "" {
		t.Errorf("empty subdomainResults.Render() = %q, want empty", got)
	}
}
