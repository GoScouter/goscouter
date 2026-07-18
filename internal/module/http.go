package module

import (
	"flag"
	"fmt"
	"io"
	"net/url"
	"strings"

	"goscouter/internal/web"

	"github.com/GoScouter/sdk"
)

type HttpModule struct{}

func (m *HttpModule) Name() string {
	return "http"
}

func (m *HttpModule) Description() string {
	return "Gather the http information of the target domain."
}

func (m *HttpModule) Version() string {
	return "0.0.1"
}

func (m *HttpModule) Scout(target string, args []string) (sdk.Result, error) {
	fs := flag.NewFlagSet("http", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	useHTTPS := fs.Bool("ssl", false, "force the https:// scheme on the target")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	fmt.Printf("» http: probing %s\r\n", target)

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

	return records, nil
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
