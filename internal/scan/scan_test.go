package scan

import (
	"strings"
	"testing"

	"goscouter/pkg/records"
)

func TestHostOf(t *testing.T) {
	cases := map[string]string{
		"https://example.com/path": "example.com",
		"http://example.com":       "example.com",
		"example.com":              "example.com",
		"example.com:8443":         "example.com",
		"example.com/a/b?q=1":      "example.com",
		"*.example.com":            "example.com",
		"  https://EXAMPLE.com  ":  "EXAMPLE.com",
	}
	for in, want := range cases {
		if got := hostOf(in); got != want {
			t.Errorf("hostOf(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestReachable(t *testing.T) {
	if (HostReport{HTTPErr: "x", HTTPSErr: "y"}).Reachable() {
		t.Error("host with only errors should not be reachable")
	}
	if !(HostReport{HTTP: &records.HTTPRecords{}}).Reachable() {
		t.Error("host with an HTTP response should be reachable")
	}
	if !(HostReport{HTTPS: &records.HTTPRecords{}}).Reachable() {
		t.Error("host with an HTTPS response should be reachable")
	}
}

func sampleGraph() *Graph {
	return &Graph{Root: &Node{
		Report: HostReport{
			Host: "example.com",
			DNS:  &records.DNSRecords{Host: "example.com", A: []string{"93.184.216.34"}},
			HTTP: &records.HTTPRecords{Status: "200 OK", Proto: "HTTP/1.1"},
		},
		Children: []*Node{
			{Report: HostReport{
				Host:  "api.example.com",
				HTTPS: &records.HTTPRecords{Status: "200 OK"},
			}},
			{Report: HostReport{
				Host:    "dead.example.com",
				HTTPErr: "connection refused",
			}},
		},
	}}
}

func TestHTMLSummary(t *testing.T) {
	html, sum := sampleGraph().HTML()

	if sum.Target != "example.com" {
		t.Errorf("Target = %q, want example.com", sum.Target)
	}
	if sum.Subdomains != 2 {
		t.Errorf("Subdomains = %d, want 2", sum.Subdomains)
	}
	if sum.Reachable != 1 {
		t.Errorf("Reachable = %d, want 1 (api reachable, dead not)", sum.Reachable)
	}

	if !strings.Contains(html, "<canvas") {
		t.Error("HTML missing <canvas>")
	}
	for _, host := range []string{"example.com", "api.example.com", "dead.example.com"} {
		if !strings.Contains(html, host) {
			t.Errorf("HTML missing host %q in embedded data", host)
		}
	}
	// The embedded JSON must not break out of the <script> block.
	if strings.Contains(html, "</script>") != true {
		t.Error("template should contain a closing </script> tag")
	}
	if idx := strings.Index(html, "const DATA ="); idx >= 0 {
		data := html[idx:]
		end := strings.Index(data, "</script>")
		if end < 0 {
			t.Fatal("no </script> after DATA")
		}
		if strings.Contains(data[:end], "</script>") {
			t.Error("embedded graph data breaks out of the <script> block")
		}
	}
}
