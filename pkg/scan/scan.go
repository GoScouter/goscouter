package scan

import (
	"encoding/json"
	"time"
)

type ModuleResult struct {
	Module string          `json:"module"`
	Data   json.RawMessage `json:"data,omitempty"`
	Error  string          `json:"error,omitempty"`
}

type HostResult struct {
	Host      string         `json:"host"`
	LastSeen  time.Time      `json:"last_seen,omitzero"`
	Addresses []string       `json:"addresses,omitempty"`
	Modules   []ModuleResult `json:"modules"`
}

type Result struct {
	Target string       `json:"target"`
	Host   string       `json:"host"`
	Found  int          `json:"found"`
	Live   int          `json:"live"`
	Hosts  []HostResult `json:"hosts"`
}
