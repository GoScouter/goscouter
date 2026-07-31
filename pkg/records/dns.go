package records

import (
	"fmt"
	"strings"
)

type DNSRecords struct {
	Host  string   `json:"host"`
	A     []string `json:"a"`
	AAAA  []string `json:"aaaa"`
	CNAME string   `json:"cname"`
	MX    []string `json:"mx"`
	NS    []string `json:"ns"`
	TXT   []string `json:"txt"`
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
