package module

import (
	"encoding/json"
	"flag"
	"io"
	"net/url"
	"strings"

	"goscouter/internal/style"
	"goscouter/internal/web"
	"goscouter/pkg/records"

	"github.com/GoScouter/sdk"
)

type HttpModule struct{}

func (m *HttpModule) Info() sdk.ModuleInfo {
	return sdk.ModuleInfo{
		Name:         "http",
		Author:       internalAuthor,
		Description:  "Gather the http information of the target domain.",
		Dependencies: make([]sdk.ModuleNamespace, 0),
	}
}

func forceScheme(target, scheme string) string {
	if u, err := url.Parse(target); err == nil && u.Host != "" {
		u.Scheme = scheme
		return u.String()
	}

	if i := strings.Index(target, "://"); i >= 0 {
		target = target[i+3:]
	}
	return scheme + "://" + target
}

func (m *HttpModule) Scout(target string, args []string) (json.RawMessage, error) {
	fs := flag.NewFlagSet("http", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	useHTTPS := fs.Bool("ssl", false, "force the https:// scheme on the target")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	scheme := "HTTP"
	if *useHTTPS {
		target = forceScheme(target, "https")
		scheme = "HTTPS"
	} else {
		target = forceScheme(target, "http")
	}

	_, err := web.CheckSiteStatus(target)
	if err != nil {
		return nil, err
	}

	records, err := web.FetchHTTPRecords(target, scheme)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(records)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (m *HttpModule) Render(raw json.RawMessage) string {
	var r records.HTTPRecords
	if err := json.Unmarshal(raw, &r); err != nil {
		return style.Failuref("http: unreadable results: %v\r\n", err)
	}

	return r.Render()
}
