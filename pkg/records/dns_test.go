package records

import (
	"strings"
	"testing"
)

func TestDNSRecordsRender(t *testing.T) {
	r := &DNSRecords{
		Host:  "example.com",
		A:     []string{"1.2.3.4", "5.6.7.8"},
		AAAA:  []string{"::1"},
		CNAME: "alias.example.com.",
		MX:    []string{"10 mail.example.com."},
		NS:    []string{"ns1.example.com.", "ns2.example.com."},
		TXT:   []string{"v=spf1 -all"},
	}

	out := r.Render()

	wantContains := []string{
		"[DNS]",
		"A      1.2.3.4",
		"A      5.6.7.8",
		"AAAA   ::1",
		"CNAME  alias.example.com.",
		"MX     10 mail.example.com.",
		"NS     ns1.example.com.",
		"NS     ns2.example.com.",
		"TXT    v=spf1 -all",
	}
	for _, w := range wantContains {
		if !strings.Contains(out, w) {
			t.Errorf("Render() missing %q\ngot:\n%s", w, out)
		}
	}
}

func TestDNSRecordsRenderOmitsEmpty(t *testing.T) {
	r := &DNSRecords{Host: "example.com", A: []string{"1.2.3.4"}}

	out := r.Render()

	for _, label := range []string{"  AAAA ", "  CNAME ", "  MX ", "  NS ", "  TXT "} {
		if strings.Contains(out, label) {
			t.Errorf("Render() should omit empty record set %q\ngot:\n%s", label, out)
		}
	}
}

func TestDNSRecordsRenderEmpty(t *testing.T) {
	r := &DNSRecords{Host: "example.com"}

	out := r.Render()

	if !strings.Contains(out, "[DNS]") {
		t.Errorf("Render() should still contain the [DNS] heading, got:\n%s", out)
	}
	if !strings.Contains(out, "[-] no records found") {
		t.Errorf("Render() should say so when there is nothing to show, got:\n%s", out)
	}
	for _, label := range []string{"  A ", "  AAAA ", "  CNAME ", "  MX ", "  NS ", "  TXT "} {
		if strings.Contains(out, label) {
			t.Errorf("Render() unexpectedly contained %q for empty records\ngot:\n%s", label, out)
		}
	}
}

func TestRecordSetSkipsEmpty(t *testing.T) {
	if lines := recordSet("A", nil); len(lines) != 0 {
		t.Errorf("recordSet with empty values returned %q, want nothing", lines)
	}
}

func TestRecordSetFormat(t *testing.T) {
	lines := recordSet("A", []string{"1.2.3.4", "5.6.7.8"})

	want := []string{"  A      1.2.3.4", "  A      5.6.7.8"}
	if len(lines) != len(want) {
		t.Fatalf("recordSet returned %d lines, want %d", len(lines), len(want))
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("recordSet line %d = %q, want %q", i, lines[i], want[i])
		}
	}
}
