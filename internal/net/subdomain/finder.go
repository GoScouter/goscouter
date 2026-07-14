package subdomain

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"goscouter/pkg/subdomains"
)

type Finder struct {
	Name  string
	Fetch func(ctx context.Context, domain string) ([]subdomains.Subdomain, error)
}

var Finders = map[string]Finder{
	"crtsh":       {Name: "crtsh", Fetch: fetchCrtSh},
	"certspotter": {Name: "certspotter", Fetch: fetchCertSpotter},
}

const TIMEOUT time.Duration = 5 * time.Second

// crt.sh timestamps carry no timezone, e.g. "2022-12-01T00:00:00".
const crtShTimeLayout = "2006-01-02T15:04:05"

func normalize(n string) string {
	n = strings.ToLower(strings.TrimSpace(n))
	n = strings.TrimPrefix(n, "*.")
	return n
}

func keepLatest(latest map[string]time.Time, name string, t time.Time) {
	name = normalize(name)
	if name == "" {
		return
	}

	if prev, ok := latest[name]; !ok || t.After(prev) {
		latest[name] = t
	}
}

func flatten(latest map[string]time.Time) []subdomains.Subdomain {
	out := make([]subdomains.Subdomain, 0, len(latest))
	for name, t := range latest {
		out = append(out, subdomains.Subdomain{Name: name, LastSeen: t})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func newRequest(ctx context.Context, rawURL string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "goscouter/1.0")
	req.Header.Set("Accept", "application/json")
	return req, nil
}

func fetchCrtSh(ctx context.Context, domain string) ([]subdomains.Subdomain, error) {
	q := url.QueryEscape("%." + domain)
	rawURL := fmt.Sprintf("https://crt.sh/?q=%s&output=json", q)

	req, err := newRequest(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("crt.sh returned status %d", resp.StatusCode)
	}

	var entries []struct {
		NameValue  string `json:"name_value"`
		CommonName string `json:"common_name"`
		NotBefore  string `json:"not_before"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("decoding crt.sh response: %w", err)
	}

	latest := make(map[string]time.Time)
	for _, e := range entries {
		t, _ := time.Parse(crtShTimeLayout, e.NotBefore)
		for _, n := range strings.Split(e.NameValue, "\n") {
			keepLatest(latest, n, t)
		}
		if e.CommonName != "" {
			keepLatest(latest, e.CommonName, t)
		}
	}
	return flatten(latest), nil
}

func fetchCertSpotter(ctx context.Context, domain string) ([]subdomains.Subdomain, error) {
	rawURL := fmt.Sprintf(
		"https://api.certspotter.com/v1/issuances?domain=%s&include_subdomains=true&expand=dns_names",
		url.QueryEscape(domain),
	)

	req, err := newRequest(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body := bufio.NewReader(resp.Body)
		line, _ := body.ReadString('\n')
		return nil, fmt.Errorf("certspotter returned status %d: %s", resp.StatusCode, strings.TrimSpace(line))
	}

	var issuances []struct {
		DNSNames  []string `json:"dns_names"`
		NotBefore string   `json:"not_before"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&issuances); err != nil {
		return nil, fmt.Errorf("decoding certspotter response: %w", err)
	}

	latest := make(map[string]time.Time)
	for _, iss := range issuances {
		t, _ := time.Parse(time.RFC3339, iss.NotBefore)
		for _, n := range iss.DNSNames {
			keepLatest(latest, n, t)
		}
	}
	return flatten(latest), nil
}

type finderResult struct {
	source string
	subs   []subdomains.Subdomain
	err    error
}

func FindAll(ctx context.Context, domain string) ([]subdomains.Subdomain, error) {
	ch := make(chan finderResult, len(Finders))
	var wg sync.WaitGroup

	for _, f := range Finders {
		wg.Add(1)
		go func(f Finder) {
			defer wg.Done()
			subs, err := f.Fetch(ctx, domain)
			ch <- finderResult{source: f.Name, subs: subs, err: err}
		}(f)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	latest := make(map[string]time.Time)
	var errs []error
	for res := range ch {
		if res.err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", res.source, res.err))
			continue
		}
		for _, s := range res.subs {
			keepLatest(latest, s.Name, s.LastSeen)
		}
	}

	subs := flatten(latest)
	if len(subs) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("all finders failed: %w", errors.Join(errs...))
	}

	return subs, nil
}
