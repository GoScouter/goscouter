package scan

import (
	"strings"
	"testing"
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
	if (HostReport{Results: []ModuleResult{{Module: "http", Err: "x"}, {Module: "dns", Err: "y"}}}).Reachable() {
		t.Error("host where every module errored should not be reachable")
	}
	if (HostReport{Results: []ModuleResult{{Module: "http"}}}).Reachable() {
		t.Error("host with only empty output should not be reachable")
	}
	if !(HostReport{Results: []ModuleResult{{Module: "http", Output: "200 OK"}}}).Reachable() {
		t.Error("host with module output should be reachable")
	}
}

func sampleGraph() *Graph {
	return &Graph{Root: &Node{
		Report: HostReport{
			Host: "example.com",
			Results: []ModuleResult{
				{Module: "dns", Output: "A 93.184.216.34"},
				{Module: "http", Output: "200 OK"},
			},
		},
		Children: []*Node{
			{Report: HostReport{
				Host:    "api.example.com",
				Results: []ModuleResult{{Module: "http", Output: "200 OK"}},
			}},
			{Report: HostReport{
				Host:    "dead.example.com",
				Results: []ModuleResult{{Module: "http", Err: "connection refused"}},
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
