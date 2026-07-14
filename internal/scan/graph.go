package scan

import "goscouter/pkg/records"

type HostReport struct {
	Host  string               `json:"host"`
	DNS   *records.DNSRecords  `json:"dns,omitempty"`
	HTTP  *records.HTTPRecords `json:"http,omitempty"`
	HTTPS *records.HTTPRecords `json:"https,omitempty"`

	DNSErr   string `json:"dnsErr,omitempty"`
	HTTPErr  string `json:"httpErr,omitempty"`
	HTTPSErr string `json:"httpsErr,omitempty"`
}

func (r HostReport) Reachable() bool {
	return r.HTTP != nil || r.HTTPS != nil
}

type Node struct {
	Report   HostReport `json:"report"`
	Children []*Node    `json:"children,omitempty"`
}

type Graph struct {
	Root *Node `json:"root"`
}
