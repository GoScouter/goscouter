package module

import (
	"fmt"
	"net/url"

	"goscouter/internal/dns"
	"goscouter/internal/web"
)

type RecordsModule struct{}

func (m *RecordsModule) Name() string {
	return "records"
}

func (m *RecordsModule) Description() string {
	return "Gather the DNS and HTTP records of the target website."
}

func (m *RecordsModule) Scout(target string) (*Records, error) {
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
