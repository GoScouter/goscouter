package module

import "testing"

func TestRecordsModuleMetadata(t *testing.T) {
	m := &RecordsModule{}
	if m.Name() != "records" {
		t.Fatalf("expected name %q, got %q", "records", m.Name())
	}
	if m.Description() == "" {
		t.Fatal("expected a non-empty description")
	}
}

func TestHostFromTarget(t *testing.T) {
	cases := []struct {
		target  string
		want    string
		wantErr bool
	}{
		{"https://example.com", "example.com", false},
		{"https://example.com/path?q=1", "example.com", false},
		{"http://example.com:8080", "example.com", false},
		{"https://sub.example.com", "sub.example.com", false},
		{"not a url", "", true},
		{"", "", true},
	}

	for _, c := range cases {
		got, err := hostFromTarget(c.target)
		if c.wantErr {
			if err == nil {
				t.Fatalf("hostFromTarget(%q): expected error, got host %q", c.target, got)
			}
			continue
		}
		if err != nil {
			t.Fatalf("hostFromTarget(%q): unexpected error: %v", c.target, err)
		}
		if got != c.want {
			t.Fatalf("hostFromTarget(%q): got %q, want %q", c.target, got, c.want)
		}
	}
}

func TestManagerRegistersRecordsModule(t *testing.T) {
	m := NewManager()
	mod, err := m.Get("records")
	if err != nil {
		t.Fatalf("expected records module registered, got error: %v", err)
	}
	if _, ok := mod.(*RecordsModule); !ok {
		t.Fatalf("expected *RecordsModule, got %T", mod)
	}

	if _, err := m.Get("nope"); err == nil {
		t.Fatal("expected error for unknown module, got nil")
	}
}
