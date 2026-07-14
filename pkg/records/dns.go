package records

type DNSRecords struct {
	Host  string
	A     []string
	AAAA  []string
	CNAME string
	MX    []string
	NS    []string
	TXT   []string
}

