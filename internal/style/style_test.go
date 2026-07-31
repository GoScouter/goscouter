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
			"✗ ":   Error("x"),
			"✓ ":   Success("x"),
			"» ":   Info("x"),
			"[+] ": Found("x"),
			"[-] ": Failure("x"),
			"[!] ": Alert("x"),
		}
		for prefix, out := range cases {
			if !strings.HasPrefix(out, prefix) {
				t.Errorf("output %q missing prefix %q", out, prefix)
			}
		}
	})
}

func TestWidthStripsANSI(t *testing.T) {
	if got := Width("\x1b[31mabc\x1b[0m"); got != 3 {
		t.Fatalf("Width of styled %q = %d, want 3", "abc", got)
	}
	if got := Width("héllo"); got != 5 {
		t.Fatalf("Width of multibyte string = %d, want 5", got)
	}
}

func TestSectionBracketsTheTitle(t *testing.T) {
	withEnabled(t, false, func() {
		body := "  A      1.2.3.4"
		out := Section("DNS", body)

		lines := strings.Split(strings.TrimSpace(out), "\r\n")
		if lines[0] != "[DNS]" {
			t.Fatalf("first line = %q, want a bracketed title", lines[0])
		}
		if lines[1] != body {
			t.Fatalf("body line = %q, want it rendered verbatim", lines[1])
		}
	})
}

func TestSectionPadsWithBlankLines(t *testing.T) {
	withEnabled(t, false, func() {
		out := Section("DNS")

		if !strings.HasPrefix(out, "\r\n") || !strings.HasSuffix(out, "\r\n\r\n") {
			t.Fatalf("Section = %q, want it padded away from the surrounding output", out)
		}
	})
}

func TestFieldPadsLabel(t *testing.T) {
	withEnabled(t, false, func() {
		if got := Field("A", 6, "1.2.3.4"); got != "  A      1.2.3.4" {
			t.Fatalf("Field = %q, want the label padded into a column", got)
		}
	})
}
