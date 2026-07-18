package records

import (
	"fmt"
	"strings"
)

type DNSRecords struct {
	Host  string
	A     []string
	AAAA  []string
	CNAME string
	MX    []string
	NS    []string
	TXT   []string
}

func (r *DNSRecords) Render() string {
	var b strings.Builder

	b.WriteString("\r\n[DNS]\r\n")
	writeRecordSet(&b, "A", r.A)
	writeRecordSet(&b, "AAAA", r.AAAA)
	if r.CNAME != "" {
		writeRecordSet(&b, "CNAME", []string{r.CNAME})
	}

    writeRecordSet(&b, "MX", r.MX)
	writeRecordSet(&b, "NS", r.NS)
	writeRecordSet(&b, "TXT", r.TXT)

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

