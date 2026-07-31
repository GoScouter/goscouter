package module

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"goscouter/internal/net/subdomain"
	pkg "goscouter/pkg/subdomains"

	"github.com/GoScouter/sdk"
)

type SubdomainsModule struct{}

func (m *SubdomainsModule) Info() sdk.ModuleInfo {
	return sdk.ModuleInfo{
		Name:         "subdomains",
		Author:       internalAuthor,
		Description:  "Gather the subdomains of the target domain.",
		Dependencies: make([]sdk.ModuleNamespace, 0),
	}
}

func (m *SubdomainsModule) Scout(target string, _ []string) (json.RawMessage, error) {
	fmt.Printf("» subdomains: enumerating %s\r\n", target)

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
		log.Print(err.Error())
		return ""
	}

	var b strings.Builder
	for _, s := range results.Subs {
		b.WriteString(s.Render())
		b.WriteString("\r\n")
	}
	return b.String()
}
