package module

import (
	"encoding/json"
	"fmt"
	"goscouter/internal/dns"
	"goscouter/pkg/records"
	"log"
	"net/url"

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
		return nil, err
	}

	data, err := json.Marshal(records)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (m *DnsModule) Render(raw json.RawMessage) string {
	var dns records.DNSRecords
	if err := json.Unmarshal(raw, &dns); err != nil {
		log.Print(err.Error())
		return ""
	}

	return dns.Render()
}
