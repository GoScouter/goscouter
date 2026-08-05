package subdomain

import (
	"context"
	"net"

	"goscouter/pkg/subdomains"
)
import "strings"

func FilterWildcard(
	ctx context.Context,
	subs []subdomains.Subdomain,
	wildcardIPs []string,
) []subdomains.Subdomain {


	result := make([]subdomains.Subdomain,0)


	for _, sub := range subs {


		ips, err :=
			net.DefaultResolver.LookupHost(
				ctx,
				sub.Name,
			)


		if err != nil {
			continue
		}


		isWildcard := false


		for _, ip := range ips {
			for _, w := range wildcardIPs {

				if ip == w {
					isWildcard = true
				}

			}
		}


		if !isWildcard {
			result = append(result, sub)
		}

	}


	return result
}

func isSubdomain(name, domain string) bool {

    name = normalize(name)
    domain = normalize(domain)

    if name == domain {
        return true
    }

    return strings.HasSuffix(
        name,
        "."+domain,
    )
}