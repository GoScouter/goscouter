package records

import (
	"goscouter/internal/style"
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

const dnsLabelWidth = 6

func (r *DNSRecords) Render() string {
	sets := []struct {
		label  string
		values []string
	}{
		{"A", r.A},
		{"AAAA", r.AAAA},
		{"CNAME", nonEmpty(r.CNAME)},
		{"MX", r.MX},
		{"NS", r.NS},
		{"TXT", r.TXT},
	}

	lines := make([]string, 0, len(sets))
	for _, set := range sets {
		lines = append(lines, recordSet(set.label, set.values)...)
	}

	if len(lines) == 0 {
		lines = append(lines, style.Failure("no records found"))
	}

	return style.Section("DNS", lines...)
}

func recordSet(label string, values []string) []string {
	lines := make([]string, 0, len(values))
	for _, v := range values {
		lines = append(lines, style.Field(label, dnsLabelWidth, style.White(v)))
	}
	return lines
}

func nonEmpty(v string) []string {
	if v == "" {
		return nil
	}
	return []string{v}
}
