package subdomain

import (
	"context"
	"net"
	"slices"
	"strings"
	"sync"

	"goscouter/pkg/subdomains"
)

const filterWorkers = 20

// FilterWildcard drops the subdomains that resolve to one of the wildcard
// addresses. Names that fail to resolve are kept, since a failed lookup says
// nothing about the wildcard.
func FilterWildcard(ctx context.Context, subs []subdomains.Subdomain, wildcardIPs []string) []subdomains.Subdomain {
	keep := make([]bool, len(subs))
	jobs := make(chan int)

	var wg sync.WaitGroup
	for range min(filterWorkers, len(subs)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				keep[i] = !isWildcardHit(ctx, subs[i].Name, wildcardIPs)
			}
		}()
	}

	for i := range subs {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	result := make([]subdomains.Subdomain, 0, len(subs))
	for i, sub := range subs {
		if keep[i] {
			result = append(result, sub)
		}
	}
	return result
}

func isWildcardHit(ctx context.Context, name string, wildcardIPs []string) bool {
	ips, err := net.DefaultResolver.LookupHost(ctx, name)
	if err != nil {
		return false
	}
	for _, ip := range ips {
		if slices.Contains(wildcardIPs, ip) {
			return true
		}
	}
	return false
}

func isSubdomain(name, domain string) bool {
	name = normalize(name)
	domain = normalize(domain)

	if name == domain {
		return true
	}
	return strings.HasSuffix(name, "."+domain)
}
