package module

import (
	"fmt"
	"net/url"

	"goscouter/internal/dns"

	"github.com/GoScouter/sdk"
)

type RecordsModule struct{}

func (m *RecordsModule) Name() string {
	return "dns"
}

func (m *RecordsModule) Description() string {
	return "Gather the DNS records of the target domain."
}

func (m *RecordsModule) Version() string {
    return "0.0.1"
}

func (m *RecordsModule) Scout(target string, _ []string) (sdk.Result, error) {
	parsed, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("invalid target %q: %w", target, err)
	}

	records, err := dns.Lookup(parsed.Path)
    if err != nil {
        return nil, err
    }

	return records, nil
}
