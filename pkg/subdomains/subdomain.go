package subdomains

import (
	"strings"
	"time"

	"github.com/GoScouter/sdk/style"
)

type Subdomain struct {
	Name     string
	LastSeen time.Time
}

func (s *Subdomain) Render() string {
	return style.Foundf("Found: %s %s",
		style.White(s.Name),
		style.Dim("("+s.LastSeen.Format(time.RFC3339)+")"),
	)
}

// RenderAll renders every subdomain, one hit per line, padded with blank lines
// so the block sits apart from the prompt like the other modules' output.
func RenderAll(subs []Subdomain) string {
	var b strings.Builder

	b.WriteString("\r\n")
	if len(subs) == 0 {
		b.WriteString(style.Failure("No subdomains found") + "\r\n")
	}
	for _, s := range subs {
		b.WriteString(s.Render())
		b.WriteString("\r\n")
	}
	b.WriteString("\r\n")

	return b.String()
}
