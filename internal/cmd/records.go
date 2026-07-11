package cmd

import (
	"fmt"
	"sort"
	"strings"

	"goscouter/internal/module"
)

type RecordsCommand struct {
	Target string
	Module module.Module
}

func (cmd *RecordsCommand) Name() string {
	return "records"
}

func (cmd *RecordsCommand) Description() string {
	return "Show the DNS and HTTP records of the target website."
}

func (cmd *RecordsCommand) Exec(args []string) error {
	target := cmd.Target
	if len(args) > 0 && args[0] != "" {
		target = args[0]
	}

	if target == "" {
		return fmt.Errorf("no target set; start gs with --target or pass one: records <url>")
	}

	records, err := cmd.Module.Scout(target)
	if err != nil {
		return err
	}

	fmt.Print(renderRecords(records))
	return nil
}

func renderRecords(r *module.Records) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Records for %s (host: %s)\r\n", r.Target, r.Host)

	b.WriteString("\r\n[DNS]\r\n")
	if r.DNS == nil {
		b.WriteString("  (no DNS records resolved)\r\n")
	} else {
		writeRecordSet(&b, "A", r.DNS.A)
		writeRecordSet(&b, "AAAA", r.DNS.AAAA)
		if r.DNS.CNAME != "" {
			writeRecordSet(&b, "CNAME", []string{r.DNS.CNAME})
		}
		writeRecordSet(&b, "MX", r.DNS.MX)
		writeRecordSet(&b, "NS", r.DNS.NS)
		writeRecordSet(&b, "TXT", r.DNS.TXT)
	}

	b.WriteString("\r\n[HTTP]\r\n")
	if r.HTTP == nil {
		b.WriteString("  (no HTTP response gathered)\r\n")
	} else {
		fmt.Fprintf(&b, "  Status   : %s\r\n", r.HTTP.Status)
		fmt.Fprintf(&b, "  Protocol : %s\r\n", r.HTTP.Proto)
		if r.HTTP.FinalURL != "" && r.HTTP.FinalURL != r.HTTP.RequestURL {
			fmt.Fprintf(&b, "  Redirect : %s -> %s\r\n", r.HTTP.RequestURL, r.HTTP.FinalURL)
		}

		b.WriteString("  Headers  :\r\n")
		keys := make([]string, 0, len(r.HTTP.Headers))
		for k := range r.HTTP.Headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		if len(keys) == 0 {
			b.WriteString("    (none)\r\n")
		}
		for _, k := range keys {
			fmt.Fprintf(&b, "    %s: %s\r\n", k, strings.Join(r.HTTP.Headers[k], ", "))
		}
	}

	b.WriteString("\r\n")
	return b.String()
}

func writeRecordSet(b *strings.Builder, label string, values []string) {
	if len(values) == 0 {
		return
	}

	for _, v := range values {
		fmt.Fprintf(b, "  %-6s %s\r\n", label, v)
	}
}
