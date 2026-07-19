package scan

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"
)

const (
	pdfMaxOutputChars = 600
)

func (g *Graph) PDF() ([]byte, Summary, error) {
	sum := Summary{}

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("GoScouter scan summary", false)
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 20)

	if g.Root == nil {
		pdf.AddPage()
		pdf.SetFont("Helvetica", "B", 16)
		pdf.CellFormat(0, 10, "GoScouter scan summary", "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 11)
		pdf.CellFormat(0, 8, "No scan data available.", "", 1, "L", false, 0, "")
		var buf bytes.Buffer
		if err := pdf.Output(&buf); err != nil {
			return nil, sum, err
		}
		return buf.Bytes(), sum, nil
	}

	sum.Target = g.Root.Report.Host
	rootReachable := g.Root.Report.Reachable()

	for _, c := range g.Root.Children {
		sum.Subdomains++
		if c.Report.Reachable() {
			sum.Reachable++
		}
	}

	hostLink := make(map[string]int, len(g.Root.Children)+1)
	hostLink[g.Root.Report.Host] = pdf.AddLink()
	for _, c := range g.Root.Children {
		hostLink[c.Report.Host] = pdf.AddLink()
	}

	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 18)
	pdf.CellFormat(0, 10, "GoScouter scan summary", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(110, 118, 129)
	pdf.CellFormat(0, 6, fmt.Sprintf("Generated %s", time.Now().Format("2006-01-02 15:04:05 MST")), "", 1, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(4)

	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(0, 8, "Overview", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 11)
	pdf.CellFormat(0, 7, fmt.Sprintf("Target: %s", sum.Target), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 7, fmt.Sprintf("Subdomains discovered: %d", sum.Subdomains), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 7, fmt.Sprintf("Reachable hosts: %d / %d", sum.Reachable, sum.Subdomains), "", 1, "L", false, 0, "")
	pdf.Ln(6)

	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(0, 8, "Table of contents", "", 1, "L", false, 0, "")
	pdf.Ln(1)

	tocEntry := func(host string, reachable bool, root bool) {
		label := host
		if root {
			label += " (root)"
		}
		pdf.SetFont("Helvetica", "", 11)
		pdf.SetTextColor(88, 166, 255)
		pdf.WriteLinkID(7, label, hostLink[host])
		pdf.SetTextColor(0, 0, 0)

		pdf.SetFont("Helvetica", "", 9)
		if reachable {
			pdf.SetTextColor(63, 185, 80)
			pdf.Write(7, "  reachable")
		} else {
			pdf.SetTextColor(248, 81, 73)
			pdf.Write(7, "  no response")
		}
		pdf.SetTextColor(0, 0, 0)
		pdf.Ln(7)
	}

	tocEntry(g.Root.Report.Host, rootReachable, true)
	for _, c := range g.Root.Children {
		tocEntry(c.Report.Host, c.Report.Reachable(), false)
	}

	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(0, 8, "Hosts", "", 1, "L", false, 0, "")
	pdf.Ln(1)

	writeHost := func(host string, reachable bool, results []ModuleResult, root bool) {
		// anchor this position as the destination for the host's TOC link
		pdf.SetLink(hostLink[host], -1, -1)
		label := host
		if root {
			label += " (root)"
		}
		pdf.Bookmark(label, 0, -1)

		pdf.SetFont("Helvetica", "B", 13)
		pdf.CellFormat(140, 8, label, "", 0, "L", false, 0, "")

		pdf.SetFont("Helvetica", "B", 10)
		if reachable {
			pdf.SetTextColor(63, 185, 80)
			pdf.CellFormat(0, 8, "reachable", "", 1, "L", false, 0, "")
		} else {
			pdf.SetTextColor(248, 81, 73)
			pdf.CellFormat(0, 8, "no response", "", 1, "L", false, 0, "")
		}
		pdf.SetTextColor(0, 0, 0)

		if len(results) == 0 {
			pdf.SetFont("Helvetica", "I", 9)
			pdf.SetTextColor(110, 118, 129)
			pdf.CellFormat(0, 6, "no module output", "", 1, "L", false, 0, "")
			pdf.SetTextColor(0, 0, 0)
		}
		for _, m := range results {
			pdf.SetFont("Helvetica", "B", 9)
			pdf.CellFormat(0, 6, "  "+m.Module, "", 1, "L", false, 0, "")
			pdf.SetFont("Helvetica", "", 9)
			if m.Err != "" {
				pdf.SetTextColor(248, 81, 73)
				pdf.MultiCell(0, 5, "  error: "+truncate(m.Err, pdfMaxOutputChars), "", "L", false)
				pdf.SetTextColor(0, 0, 0)
			} else if m.Output != "" {
				pdf.SetTextColor(60, 60, 60)
				pdf.MultiCell(0, 5, "  "+truncate(m.Output, pdfMaxOutputChars), "", "L", false)
				pdf.SetTextColor(0, 0, 0)
			}
		}
		pdf.Ln(2)
	}

	writeHost(g.Root.Report.Host, rootReachable, g.Root.Report.Results, true)
	for _, c := range g.Root.Children {
		writeHost(c.Report.Host, c.Report.Reachable(), c.Report.Results, false)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, sum, err
	}
	return buf.Bytes(), sum, nil
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}
