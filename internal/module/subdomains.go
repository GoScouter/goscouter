package module

import (
	"context"
	"encoding/json"

	"goscouter/internal/net/subdomain"
	pkg "goscouter/pkg/subdomains"

	"github.com/GoScouter/sdk"
	"github.com/GoScouter/sdk/style"
)

type SubdomainsModule struct{}

var SdModuleInfo = sdk.ModuleInfo{
	Name:         "subdomains",
	Author:       internalAuthor,
	Description:  "Gather the subdomains of the target domain.",
	Dependencies: make([]sdk.ModuleNamespace, 0),
}

func (m *SubdomainsModule) Info() sdk.ModuleInfo {
	return SdModuleInfo
}

func (m *SubdomainsModule) Scout(target string, _ []string) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), subdomain.TIMEOUT)
	defer cancel()

	subdomains, err := subdomain.FindAll(ctx, target)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(subdomainResults{Subs: subdomains})
	if err != nil {
		return nil, err
	}

	return data, nil
}

type subdomainResults struct {
	Subs []pkg.Subdomain `json:"subs"`
}

func (m *SubdomainsModule) Render(raw json.RawMessage) string {
	var results subdomainResults
	if err := json.Unmarshal(raw, &results); err != nil {
		return style.Failuref("subdomains: unreadable results: %v\r\n", err)
	}

	return pkg.RenderAll(results.Subs)
}
