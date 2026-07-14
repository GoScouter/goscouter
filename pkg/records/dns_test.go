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
		t.Errorf("Render() should still contain the [DNS] header, got:\n%s", out)
	}
	for _, label := range []string{"  A ", "  AAAA ", "  CNAME ", "  MX ", "  NS ", "  TXT "} {
		if strings.Contains(out, label) {
			t.Errorf("Render() unexpectedly contained %q for empty records\ngot:\n%s", label, out)
		}
	}
}

func TestWriteRecordSetSkipsEmpty(t *testing.T) {
	var b strings.Builder
	writeRecordSet(&b, "A", nil)
	if b.Len() != 0 {
		t.Errorf("writeRecordSet with empty values wrote %q, want nothing", b.String())
	}
}

func TestWriteRecordSetFormat(t *testing.T) {
	var b strings.Builder
	writeRecordSet(&b, "A", []string{"1.2.3.4"})
	want := "  A      1.2.3.4\r\n"
	if b.String() != want {
		t.Errorf("writeRecordSet = %q, want %q", b.String(), want)
	}
}
