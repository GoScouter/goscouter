package records

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

type HTTPRecords struct {
	Scheme     string
    RequestURL string
	FinalURL   string
	StatusCode int
	Status     string
	Proto      string
	Headers    http.Header
}

func (r *HTTPRecords) Render() string {
	var b strings.Builder

	fmt.Fprintf(&b, "\r\n[%s]\r\n", r.Scheme)
    fmt.Fprintf(&b, "  Status   : %s\r\n", r.Status)
	fmt.Fprintf(&b, "  Protocol : %s\r\n", r.Proto)
	if r.FinalURL != "" && r.FinalURL != r.RequestURL {
		fmt.Fprintf(&b, "  Redirect : %s -> %s\r\n", r.RequestURL, r.FinalURL)
	}

	b.WriteString("  Headers  :\r\n")
	keys := make([]string, 0, len(r.Headers))
	for k := range r.Headers {
		keys = append(keys, k)
	}

    sort.Strings(keys)
	if len(keys) == 0 {
		b.WriteString("    (none)\r\n")
	}

    for _, k := range keys {
		fmt.Fprintf(&b, "    %s: %s\r\n", k, strings.Join(r.Headers[k], ", "))
	}

	b.WriteString("\r\n")
	return b.String()
}
