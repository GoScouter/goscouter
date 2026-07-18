package scan

type ModuleResult struct {
	Module string `json:"module"`
	Output string `json:"output,omitempty"`
	Err    string `json:"err,omitempty"`
}

type HostReport struct {
	Host    string         `json:"host"`
	Results []ModuleResult `json:"results,omitempty"`
}

func (r HostReport) Reachable() bool {
	for _, m := range r.Results {
		if m.Err == "" && m.Output != "" {
			return true
		}
	}
	return false
}

type Node struct {
	Report   HostReport `json:"report"`
	Children []*Node    `json:"children,omitempty"`
}

type Graph struct {
	Root *Node `json:"root"`
}
