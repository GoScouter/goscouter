package subdomain

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"sort"
	"strings"
	"time"
)

const (
	wildcardTests = 5
	wildcardLabel = 12
)

func randomLabel() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"

	r := rand.New(
		rand.NewSource(time.Now().UnixNano()),
	)

	b := make([]byte, wildcardLabel)

	for i := range b {
		b[i] = chars[r.Intn(len(chars))]
	}

	return string(b)
}


func normalizeIPs(ips []string) []string {

	out := make([]string, 0, len(ips))

	for _, ip := range ips {
		out = append(out, strings.TrimSpace(ip))
	}

	sort.Strings(out)

	return out
}


func sameIPs(a, b []string) bool {

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}


func DetectWildcard(
	ctx context.Context,
	domain string,
) ([]string, bool) {


	var baseline []string


	matches := 0


	for i := 0; i < wildcardTests; i++ {

		randomHost :=
			fmt.Sprintf(
				"%s.%s",
				randomLabel(),
				domain,
			)


		ips, err :=
			net.DefaultResolver.LookupHost(
				ctx,
				randomHost,
			)


		fmt.Println(
			"[wildcard test]",
			randomHost,
			ips,
			err,
		)


		if err != nil || len(ips) == 0 {
			continue
		}


		ips = normalizeIPs(ips)


		if baseline == nil {
			baseline = ips
			matches++
			continue
		}


		if sameIPs(
			baseline,
			ips,
		) {
			matches++
		}
	}


	// Require at least 3 random names to resolve identically
	if matches >= 3 {

		return baseline, true
	}


	return nil, false
}