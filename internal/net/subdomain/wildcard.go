package subdomain

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"slices"
	"sync"
)

const (
	wildcardTests = 5
	wildcardLabel = 12
)

func randomLabel() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"

	b := make([]byte, wildcardLabel)
	for i := range b {
		b[i] = chars[rand.IntN(len(chars))]
	}
	return string(b)
}

// DetectWildcard resolves a handful of random names under domain. If enough of
// them resolve to the same addresses, the domain has wildcard DNS and those
// addresses are returned.
func DetectWildcard(ctx context.Context, domain string) ([]string, bool) {
	results := make([][]string, wildcardTests)

	var wg sync.WaitGroup
	for i := range wildcardTests {
		wg.Add(1)
		go func() {
			defer wg.Done()
			host := fmt.Sprintf("%s.%s", randomLabel(), domain)
			ips, err := net.DefaultResolver.LookupHost(ctx, host)
			if err != nil || len(ips) == 0 {
				return
			}
			slices.Sort(ips)
			results[i] = ips
		}()
	}
	wg.Wait()

	var baseline []string
	matches := 0
	for _, ips := range results {
		if ips == nil {
			continue
		}
		if baseline == nil {
			baseline = ips
		}
		if slices.Equal(baseline, ips) {
			matches++
		}
	}

	// Require at least 3 random names to resolve identically.
	if matches >= 3 {
		return baseline, true
	}
	return nil, false
}
