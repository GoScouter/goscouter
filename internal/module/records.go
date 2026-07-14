package module

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"goscouter/internal/dns"
	"goscouter/internal/web"
	"goscouter/pkg/records"

    "github.com/GoScouter/sdk"
)

type RecordsModule struct{}

func (m *RecordsModule) Name() string {
	return "records"
}

func (m *RecordsModule) Description() string {
	return "Gather the DNS and HTTP records of the target website."
}

func (m *RecordsModule) Version() string {
    return "0.0.1"
}

func (m *RecordsModule) Scout(target string) (sdk.Result, error) {
	host, err := hostFromTarget(target)
	if err != nil {
		return nil, err
	}

	records := &Records{Target: target, Host: host}

	dnsRecords, dnsErr := dns.Lookup(host)
	if dnsErr == nil {
		records.DNS = dnsRecords
	}

	httpRecords, httpErr := web.FetchHTTPRecords(target)
	if httpErr == nil {
		records.HTTP = httpRecords
	}

	if dnsErr != nil && httpErr != nil {
		return nil, fmt.Errorf("could not gather records for %s: dns: %v; http: %v", target, dnsErr, httpErr)
	}

	return records, nil
}

type Records struct {
	Target string
	Host   string
	DNS    *records.DNSRecords
	HTTP   *records.HTTPRecords
}

func (r *Records) Render() string {
	var b strings.Builder

	fmt.Fprintf(&b, "Records for %s (host: %s)\r\n", r.Target, r.Host)

	b.WriteString("\r\n[DNS]\r\n")
	if r.DNS == nil {
		b.WriteString("  (no DNS records resolved)\r\n")
	} else {
		writeRecordSet(&b, "A", r.DNS.A)
		writeRecordSet(&b, "AAAA", r.DNS.AAAA)
		if r.DNS.CNAME != "" {
			writeRecordSet(&b, "CNAME", []string{r.DNS.CNAME})
		}
		writeRecordSet(&b, "MX", r.DNS.MX)
		writeRecordSet(&b, "NS", r.DNS.NS)
		writeRecordSet(&b, "TXT", r.DNS.TXT)
	}

	b.WriteString("\r\n[HTTP]\r\n")
	if r.HTTP == nil {
		b.WriteString("  (no HTTP response gathered)\r\n")
	} else {
		fmt.Fprintf(&b, "  Status   : %s\r\n", r.HTTP.Status)
		fmt.Fprintf(&b, "  Protocol : %s\r\n", r.HTTP.Proto)
		if r.HTTP.FinalURL != "" && r.HTTP.FinalURL != r.HTTP.RequestURL {
			fmt.Fprintf(&b, "  Redirect : %s -> %s\r\n", r.HTTP.RequestURL, r.HTTP.FinalURL)
		}

		b.WriteString("  Headers  :\r\n")
		keys := make([]string, 0, len(r.HTTP.Headers))
		for k := range r.HTTP.Headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		if len(keys) == 0 {
			b.WriteString("    (none)\r\n")
		}
		for _, k := range keys {
			fmt.Fprintf(&b, "    %s: %s\r\n", k, strings.Join(r.HTTP.Headers[k], ", "))
		}
	}

	b.WriteString("\r\n")
	return b.String()
}

func writeRecordSet(b *strings.Builder, label string, values []string) {
	if len(values) == 0 {
		return
	}

	for _, v := range values {
		fmt.Fprintf(b, "  %-6s %s\r\n", label, v)
	}
}

func hostFromTarget(target string) (string, error) {
	parsed, err := url.Parse(target)
	if err != nil {
		return "", fmt.Errorf("invalid target %q: %w", target, err)
	}

	host := parsed.Hostname()
	if host == "" {
		return "", fmt.Errorf("could not determine host from target %q", target)
	}

	return host, nil
}
