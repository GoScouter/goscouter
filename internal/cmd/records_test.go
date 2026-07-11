package cmd

import (
	"errors"
	"strings"
	"testing"

	"goscouter/internal/dns"
	"goscouter/internal/module"
	"goscouter/internal/web"
)

type fakeModule struct {
	records *module.Records
	err     error
	gotArg  string
}

func (m *fakeModule) Name() string        { return "records" }
func (m *fakeModule) Description() string  { return "fake" }
func (m *fakeModule) Scout(target string) (*module.Records, error) {
	m.gotArg = target
	return m.records, m.err
}

func TestRecordsCommandMetadata(t *testing.T) {
	c := &RecordsCommand{}
	if c.Name() != "records" {
		t.Fatalf("expected name %q, got %q", "records", c.Name())
	}
	if c.Description() == "" {
		t.Fatal("expected a non-empty description")
	}
}

func TestRecordsCommandExecRenders(t *testing.T) {
	fake := &fakeModule{
		records: &module.Records{
			Target: "https://example.com",
			Host:   "example.com",
			DNS: &dns.Records{
				Host: "example.com",
				A:    []string{"93.184.216.34"},
				MX:   []string{"10 mail.example.com."},
			},
			HTTP: &web.HTTPRecords{
				RequestURL: "https://example.com",
				FinalURL:   "https://example.com",
				StatusCode: 200,
				Status:     "200 OK",
				Proto:      "HTTP/2.0",
				Headers:    map[string][]string{"Server": {"ECS"}},
			},
		},
	}

	c := &RecordsCommand{Target: "https://example.com", Module: fake}
	out := captureStdout(t, func() {
		if err := c.Exec(nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, want := range []string{"example.com", "93.184.216.34", "mail.example.com", "200 OK", "Server", "ECS"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected records output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestRecordsCommandExecUsesArgOverTarget(t *testing.T) {
	fake := &fakeModule{records: &module.Records{Target: "https://other.com", Host: "other.com"}}
	c := &RecordsCommand{Target: "https://example.com", Module: fake}

	_ = captureStdout(t, func() {
		if err := c.Exec([]string{"https://other.com"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if fake.gotArg != "https://other.com" {
		t.Fatalf("expected the argument target to be used, got %q", fake.gotArg)
	}
}

func TestRecordsCommandExecNoTarget(t *testing.T) {
	c := &RecordsCommand{Module: &fakeModule{}}
	if err := c.Exec(nil); err == nil {
		t.Fatal("expected an error when no target is set, got nil")
	}
}

func TestRecordsCommandExecPropagatesModuleError(t *testing.T) {
	wantErr := errors.New("boom")
	c := &RecordsCommand{Target: "https://example.com", Module: &fakeModule{err: wantErr}}

	err := c.Exec(nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected module error to propagate, got %v", err)
	}
}
