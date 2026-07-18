package subdomains

import (
	"fmt"
	"time"
)

type Subdomain struct {
	Name     string
	LastSeen time.Time
}

func (s *Subdomain) Render() string {
    return fmt.Sprintf("[+] Found: %s (%s)", s.Name, s.LastSeen.Format(time.RFC3339))
}
