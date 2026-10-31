package cmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"goscouter/internal/module"
)

func docsManager() *CommandManager {
	cm := &CommandManager{Commands: make(map[string]Command), Target: "example.com"}
	cm.addCommand(&HelpCommand{Manager: cm})
	cm.addCommand(&TargetCommand{Manager: cm})
	cm.addCommand(&SaveCommand{Manager: cm})
	cm.addCommand(&InternalCommand{CmdManager: cm, Module: &module.HttpModule{}})
	cm.addCommand(&InternalCommand{CmdManager: cm, Module: &module.DnsModule{}})
	return cm
}

func TestCompleteArguments(t *testing.T) {
	cm := docsManager()
	cm.History = func() []string {
		return []string{"target old.com", "dns", "target websec.co.il"}
	}

	cases := []struct {
		line string
		want []string
	}{
		{"http ", []string{"--ssl"}},
		{"http --", []string{"--ssl"}},
		{"http --ssl ", nil},
		{"internal:http -", []string{"--ssl"}},
		{"dns ", nil},
		{"target ", []string{"websec.co.il", "old.com", "example.com"}},
		{"target web", []string{"websec.co.il"}},
		{"help ht", []string{"http"}},
		{"save ", []string{"results.json", "results.md", "results.txt"}},
		{"save scan.m", []string{"scan.md"}},
		{"nope x", nil},
	}
	for _, c := range cases {
		if got := cm.Complete(c.line); !slices.Equal(got, c.want) {
			t.Errorf("Complete(%q) = %v, want %v", c.line, got, c.want)
		}
	}
}

func TestHint(t *testing.T) {
	cm := docsManager()
	cases := map[string]string{
		"http":       "internal:http [--ssl]",
		"http --ssl": "internal:http [--ssl]",
		"target x":   "target [<example.com>]",
		"nope":       "",
		"":           "",
		"!ls":        "",
	}
	for line, want := range cases {
		got := cm.Hint(line)
		if want == "" && got != "" {
			t.Errorf("Hint(%q) = %q, want none", line, got)
		}
		if want != "" && !strings.HasPrefix(got, want) {
			t.Errorf("Hint(%q) = %q, want it to start with %q", line, got, want)
		}
	}
}

func TestHelpDescribesCommand(t *testing.T) {
	cm := docsManager()
	help := cm.Commands["help"]

	out := captureStdout(t, func() {
		if err := help.Exec(context.Background(), []string{"http"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	for _, want := range []string{"internal:http", "Usage", "internal:http [--ssl]", "or just http", "--ssl", "https://"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in:\n%s", want, out)
		}
	}

	if err := help.Exec(context.Background(), []string{"nope"}); err == nil {
		t.Fatal("expected an error for an unknown command")
	}
}

func TestHelpListsKeys(t *testing.T) {
	cm := docsManager()
	out := captureStdout(t, func() {
		_ = cm.Commands["help"].Exec(context.Background(), nil)
	})
	for _, want := range []string{"Keys", "Ctrl-R", "Ctrl-F", "Ctrl-C"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in:\n%s", want, out)
		}
	}
}

func TestSaveNothingYet(t *testing.T) {
	cm := docsManager()
	if err := cm.Commands["save"].Exec(context.Background(), []string{"x.json"}); err == nil {
		t.Fatal("expected an error before any result exists")
	}
}

func TestSaveFormats(t *testing.T) {
	cm := docsManager()
	cm.Last = &Result{
		Command: "internal:dns",
		Target:  "example.com",
		Time:    time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC),
		Data:    []json.RawMessage{json.RawMessage(`{"A":["1.2.3.4"]}`)},
		Views:   []string{"\x1b[36m[DNS]\x1b[0m\r\n  A  1.2.3.4\r\n"},
	}
	dir := t.TempDir()

	cases := map[string][]string{
		"out.json": {`"A": [`, `"1.2.3.4"`},
		"out.md":   {"# goscouter: internal:dns", "**Target:** example.com", "```text", "[DNS]\n  A  1.2.3.4"},
		"out.txt":  {"[DNS]\n  A  1.2.3.4"},
	}
	for name, wants := range cases {
		path := filepath.Join(dir, name)
		captureStdout(t, func() {
			if err := cm.Commands["save"].Exec(context.Background(), []string{path}); err != nil {
				t.Fatalf("save %s: %v", name, err)
			}
		})

		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range wants {
			if !strings.Contains(string(content), want) {
				t.Errorf("%s: expected %q in:\n%s", name, want, content)
			}
		}
		if strings.Contains(string(content), "\x1b") {
			t.Errorf("%s: colors must be stripped, got %q", name, content)
		}
	}
}

func TestSaveJSONIsValid(t *testing.T) {
	r := &Result{Data: []json.RawMessage{json.RawMessage(`{"a":1}`), json.RawMessage(`{"b":2}`)}}
	out, err := resultJSON(r)
	if err != nil {
		t.Fatal(err)
	}
	var parsed []map[string]int
	if err := json.Unmarshal(out, &parsed); err != nil || len(parsed) != 2 {
		t.Fatalf("expected a JSON array of both results, got %s (%v)", out, err)
	}
}
