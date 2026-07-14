package scan

import (
	"context"
	"net/url"
	"sort"
	"strings"
	"sync"

	"goscouter/internal/net/subdomain"

	"github.com/GoScouter/sdk"
)

// maxConcurrentProbes bounds how many hosts are probed at once. There is no cap
// on how many subdomains get scanned, only on simultaneous in-flight hosts so
// a domain with hundreds of subdomains does not spawn hundreds of module runs
// at once. Modules run sequentially within a single host.
const maxConcurrentProbes = 12

func hostOf(target string) string {
	t := strings.TrimSpace(target)
	t = strings.TrimPrefix(t, "*.")

	if strings.Contains(t, "://") {
		if u, err := url.Parse(t); err == nil && u.Hostname() != "" {
			return u.Hostname()
		}
	}

	// No scheme: url.Parse would shove everything into Path, so parse by hand.
	if i := strings.IndexAny(t, "/?#"); i >= 0 {
		t = t[:i]
	}
	if h, _, ok := strings.Cut(t, ":"); ok {
		t = h
	}
	return t
}

func Build(ctx context.Context, target string, mods []sdk.Module) (*Graph, error) {
	host := hostOf(target)

	subs, _ := subdomain.FindAll(ctx, host)
	root := &Node{Report: probeHost(host, mods)}

	sem := make(chan struct{}, maxConcurrentProbes)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, s := range subs {
		name := hostOf(s.Name)
		if name == "" || name == host {
			continue
		}

		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			child := &Node{Report: probeHost(name, mods)}
			mu.Lock()
			root.Children = append(root.Children, child)
			mu.Unlock()
		}(name)
	}
	wg.Wait()

	sort.Slice(root.Children, func(i, j int) bool {
		return root.Children[i].Report.Host < root.Children[j].Report.Host
	})

	return &Graph{Root: root}, nil
}

func probeHost(host string, mods []sdk.Module) HostReport {
	report := HostReport{Host: host}

	for _, mod := range mods {
		result := ModuleResult{Module: mod.Name()}
		if res, err := mod.Scout(host, nil); err != nil {
			result.Err = err.Error()
		} else {
			result.Output = res.Render()
		}
		report.Results = append(report.Results, result)
	}

	sort.Slice(report.Results, func(i, j int) bool {
		return report.Results[i].Module < report.Results[j].Module
	})

	return report
}
