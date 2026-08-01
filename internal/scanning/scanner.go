package scanning

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"sync"
	"time"

	"goscouter/internal/dns"
	"goscouter/internal/logging"
	"goscouter/internal/module"
	"goscouter/internal/net/subdomain"
	"goscouter/internal/style"
	"goscouter/pkg/subdomains"
)

const (
	maxConcurrentProbes  = 32
	maxConcurrentLookups = 64
)

type ModuleResult struct {
	Module string          `json:"module"`
	Data   json.RawMessage `json:"data,omitempty"`
	Error  string          `json:"error,omitempty"`
}

type HostResult struct {
	Host      string         `json:"host"`
	LastSeen  time.Time      `json:"last_seen,omitzero"`
	Addresses []string       `json:"addresses,omitempty"`
	Modules   []ModuleResult `json:"modules"`
}

type Result struct {
	Target string       `json:"target"`
	Host   string       `json:"host"`
	Found  int          `json:"found"`
	Live   int          `json:"live"`
	Hosts  []HostResult `json:"hosts"`
}

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

func Scan(target string, manager *module.Manager) (json.RawMessage, error) {
	if manager == nil || manager.Graph == nil {
		return nil, errors.New("scanning: module graph is not built")
	}

	started := time.Now()
	host := hostOf(target)

	order, err := manager.Graph.Order()
	if err != nil {
		return nil, err
	}

	logging.Step("Enumerating subdomains of %s", style.Bold(host))

	ctx, cancel := context.WithTimeout(context.Background(), subdomain.TIMEOUT)
	defer cancel()

	found, err := subdomain.FindAll(ctx, target)
	if err != nil {
		return nil, err
	}

	candidates := candidates(found, host)
	logging.Step("%d unique subdomains, resolving", len(candidates))

	hosts := resolve(context.Background(), candidates, order)

	result := Result{
		Target: target,
		Host:   host,
		Found:  len(candidates),
		Live:   len(hosts),
		Hosts:  hosts,
	}

	if len(hosts) == 0 {
		logging.Alert("No live hosts to probe")
		return json.Marshal(result)
	}

	malshan, err := module.DialReporter()
	if err != nil {
		return nil, err
	}
	defer malshan.Close()

	sem := make(chan struct{}, maxConcurrentProbes)

	for mi, m := range order {
		logging.Step("Running %s against %d hosts", style.Bold(m), len(result.Hosts))

		var wg sync.WaitGroup

		for hi := range result.Hosts {
			wg.Add(1)
			go func(slot *ModuleResult, name string, m string, reporter *module.Reporter) {
				defer wg.Done()

				sem <- struct{}{}
				defer func() { <-sem }()

				slot.Module = m

				data, _, err := module.RunModule(name, []string{}, m, manager, reporter)
				if err != nil {
					slot.Error = err.Error()
					logging.Failed("%s %s: %v", pad(name), style.Gray(m), err)
					return
				}

				slot.Data = data

				if len(data) == 0 || string(data) == "null" {
					logging.Alert("%s %s: nothing to report", pad(name), style.Gray(m))
					return
				}

				logging.Found("%s %s", pad(name), style.Gray(m))
			}(&result.Hosts[hi].Modules[mi], result.Hosts[hi].Host, m, malshan)
		}

		wg.Wait()
	}

	logging.Step("Scan finished in %s", time.Since(started).Round(time.Millisecond))

	return json.Marshal(result)
}

func candidates(found []subdomains.Subdomain, host string) []subdomains.Subdomain {
	out := make([]subdomains.Subdomain, 0, len(found))
	seen := make(map[string]bool, len(found))

	for _, sub := range found {
		name := hostOf(sub.Name)
		if name == "" || name == host || seen[name] {
			continue
		}
		seen[name] = true

		out = append(out, subdomains.Subdomain{Name: name, LastSeen: sub.LastSeen})
	}

	return out
}

func resolve(ctx context.Context, candidates []subdomains.Subdomain, order []string) []HostResult {
	type lookup struct {
		addrs []string
		err   error
	}

	results := make([]lookup, len(candidates))

	sem := make(chan struct{}, maxConcurrentLookups)
	var wg sync.WaitGroup

	for i, c := range candidates {
		wg.Add(1)
		go func(slot *lookup, name string) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			slot.addrs, slot.err = dns.Resolve(ctx, name)
		}(&results[i], c.Name)
	}

	wg.Wait()

	hosts := make([]HostResult, 0, len(candidates))
	for i, c := range candidates {
		if results[i].err != nil {
			logging.Failed("%s does not resolve", pad(c.Name))
			continue
		}

		logging.Found("%s %s", pad(c.Name), style.Gray(strings.Join(results[i].addrs, ", ")))

		hosts = append(hosts, HostResult{
			Host:      c.Name,
			LastSeen:  c.LastSeen,
			Addresses: results[i].addrs,
			Modules:   make([]ModuleResult, len(order)),
		})
	}

	return hosts
}

func pad(name string) string {
	if len(name) >= 38 {
		return style.White(name)
	}
	return style.White(name) + strings.Repeat(" ", 38-len(name))
}
