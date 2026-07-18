package module

import (
	"context"
	"fmt"
	"strings"

	"goscouter/internal/net/subdomain"
	pkg "goscouter/pkg/subdomains"

	"github.com/GoScouter/sdk"
)

type SubdomainsModule struct{}

func (m *SubdomainsModule) Name() string {
	return "subdomains"
}

func (m *SubdomainsModule) Description() string {
	return "Gather the subdomains of the target domain."
}

func (m *SubdomainsModule) Version() string {
	return "0.0.1"
}

type finderResult struct {
	source string
	names  []string
	err    error
}

type subdomainResults struct {
	Subs []pkg.Subdomain
}

func (r subdomainResults) Render() string {
	var b strings.Builder
	for _, s := range r.Subs {
		b.WriteString(s.Render())
		b.WriteString("\r\n")
	}
	return b.String()
}

func (m *SubdomainsModule) Scout(target string, _ []string) (sdk.Result, error) {
	fmt.Printf("» subdomains: enumerating %s\r\n", target)

	ctx, cancel := context.WithTimeout(context.Background(), subdomain.TIMEOUT)
	defer cancel()

	subdomains, err := subdomain.FindAll(ctx, target)
	if err != nil {
		return nil, err
	}

	return subdomainResults{Subs: subdomains}, nil
}
