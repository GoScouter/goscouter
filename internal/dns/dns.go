package dns

import (
	"context"
	"fmt"
	"net"
	"time"

    "goscouter/pkg/records"
)

const TIMEOUT time.Duration = 5 * time.Second

func Lookup(host string) (*records.DNSRecords, error) {
	ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT)
	defer cancel()

	resolver := net.DefaultResolver
	records := &records.DNSRecords{Host: host}

	ips, ipErr := resolver.LookupIP(ctx, "ip", host)
	for _, ip := range ips {
		if ip.To4() != nil {
			records.A = append(records.A, ip.String())
		} else {
			records.AAAA = append(records.AAAA, ip.String())
		}
	}

	if len(ips) == 0 && ipErr != nil {
		return nil, fmt.Errorf("could not resolve %s: %w", host, ipErr)
	}

	if cname, err := resolver.LookupCNAME(ctx, host); err == nil && cname != host+"." {
		records.CNAME = cname
	}

	if mx, err := resolver.LookupMX(ctx, host); err == nil {
		for _, m := range mx {
			records.MX = append(records.MX, fmt.Sprintf("%d %s", m.Pref, m.Host))
		}
	}

	if ns, err := resolver.LookupNS(ctx, host); err == nil {
		for _, n := range ns {
			records.NS = append(records.NS, n.Host)
		}
	}

	if txt, err := resolver.LookupTXT(ctx, host); err == nil {
		records.TXT = txt
	}

	return records, nil
}
