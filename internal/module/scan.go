package module

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"goscouter/internal/logger"
	"goscouter/internal/scan"

	"github.com/GoScouter/sdk"
)

type ScanModule struct {
	Manager *Manager
}

var scanExcluded = map[string]bool{
	"subdomains": true,
	"scan":       true,
}

func (m *ScanModule) modulesForScan() ([]sdk.Module, func()) {
	var mods []sdk.Module

	if m.Manager != nil {
		for _, mod := range m.Manager.GetAll() {
			if !scanExcluded[mod.Name()] {
				mods = append(mods, mod)
			}
		}
	}

	external, cleanup, err := LoadExternal()
	if err != nil {
		logger.Log.Warn(fmt.Sprintf("scan: loading external modules: %v", err))
	}
	for _, mod := range external {
		if !scanExcluded[mod.Name()] {
			mods = append(mods, mod)
		}
	}

	return mods, cleanup
}

func (m *ScanModule) Name() string {
	return "scan"
}

func (m *ScanModule) Description() string {
	return "Crawl the target and its subdomains, then render a spider-web graph of DNS/HTTP findings to an HTML page and a PDF summary."
}

func (m *ScanModule) Version() string {
	return "0.0.1"
}

type scanResult struct {
	summary scan.Summary
	path    string
	pdfPath string
}

func (r scanResult) Render() string {
	var b strings.Builder
	b.WriteString("\r\n[SCAN]\r\n")
	fmt.Fprintf(&b, "  Target      : %s\r\n", r.summary.Target)
	fmt.Fprintf(&b, "  Subdomains  : %d discovered\r\n", r.summary.Subdomains)
	fmt.Fprintf(&b, "  Reachable   : %d\r\n", r.summary.Reachable)
	fmt.Fprintf(&b, "  Graph       : %s\r\n", r.path)
	fmt.Fprintf(&b, "  PDF summary : %s\r\n", r.pdfPath)
	b.WriteString("\r\n")
	return b.String()
}

func (m *ScanModule) Scout(target string, args []string) (sdk.Result, error) {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	out := fs.String("out", "", "path for the generated HTML graph (default gs-scan-<host>.html)")
	pdfOut := fs.String("pdf-out", "", "path for the generated PDF summary (default gs-scan-<host>.pdf)")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	mods, cleanup := m.modulesForScan()
	defer cleanup()

	graph, err := scan.Build(context.Background(), target, mods)
	if err != nil {
		return nil, err
	}

	html, summary := graph.HTML()
	path := *out
	if path == "" {
		path = fmt.Sprintf("gs-scan-%s.html", summary.Target)
	}

	if err := os.WriteFile(path, []byte(html), 0o644); err != nil {
		return nil, fmt.Errorf("writing graph to %s: %w", path, err)
	}

	pdf, _, err := graph.PDF()
	if err != nil {
		return nil, fmt.Errorf("generating PDF summary: %w", err)
	}
	pdfPath := *pdfOut
	if pdfPath == "" {
		pdfPath = fmt.Sprintf("gs-scan-%s.pdf", summary.Target)
	}

	if err := os.WriteFile(pdfPath, pdf, 0o644); err != nil {
		return nil, fmt.Errorf("writing PDF summary to %s: %w", pdfPath, err)
	}

	return scanResult{summary: summary, path: path, pdfPath: pdfPath}, nil
}
