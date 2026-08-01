package module

import (
	"encoding/json"
	"fmt"
	"net/url"

	"goscouter/internal/dns"
	"goscouter/internal/logging"
	"goscouter/internal/style"
	"goscouter/pkg/records"

	"github.com/GoScouter/sdk"
)

type DnsModule struct{}

func (m *DnsModule) Info() sdk.ModuleInfo {
	return sdk.ModuleInfo{
		Name:         "dns",
		Author:       internalAuthor,
		Description:  "Gather the DNS records of the target domain.",
		Dependencies: make([]sdk.ModuleNamespace, 0),
	}
}

func (m *DnsModule) Scout(target string, _ []string) (json.RawMessage, error) {
	parsed, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("invalid target %q: %w", target, err)
	}

	records, err := dns.Lookup(parsed.Path)
	if err != nil {
		logging.Failed("dns: %s: %v", target, err)
		return nil, err
	}

	logging.Found("dns: %s (%d A, %d AAAA, %d MX, %d NS, %d TXT)",
		target, len(records.A), len(records.AAAA), len(records.MX), len(records.NS), len(records.TXT))

	data, err := json.Marshal(records)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (m *DnsModule) Render(raw json.RawMessage) string {
	var dns records.DNSRecords
	if err := json.Unmarshal(raw, &dns); err != nil {
		return style.Failuref("dns: unreadable results: %v\r\n", err)
	}

	return dns.Render()
}
