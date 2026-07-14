package scan

import (
	"context"
	"net/url"
	"sort"
	"strings"
	"sync"

	"goscouter/internal/dns"
	"goscouter/internal/net/subdomain"
	"goscouter/internal/web"
)

// maxConcurrentProbes bounds how many hosts are probed at once. There is no cap
// on how many subdomains get scanned, only on simultaneous in-flight sockets so
// a domain with hundreds of subdomains does not open hundreds of connections at
// once.
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

func Build(ctx context.Context, target string) (*Graph, error) {
	host := hostOf(target)

	subs, _ := subdomain.FindAll(ctx, host)
	root := &Node{Report: probeHost(host)}

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

			child := &Node{Report: probeHost(name)}
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

func probeHost(host string) HostReport {
	report := HostReport{Host: host}

	if rec, err := dns.Lookup(host); err != nil {
		report.DNSErr = err.Error()
	} else {
		report.DNS = rec
	}

	if rec, err := web.FetchHTTPRecords("http://"+host, "HTTP"); err != nil {
		report.HTTPErr = err.Error()
	} else {
		report.HTTP = rec
	}

	if rec, err := web.FetchHTTPRecords("https://"+host, "HTTPS"); err != nil {
		report.HTTPSErr = err.Error()
	} else {
		report.HTTPS = rec
	}

	return report
}
