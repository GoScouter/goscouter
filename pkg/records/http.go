package records

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/GoScouter/sdk/style"
)

type HTTPRecords struct {
	Scheme     string      `json:"scheme"`
	RequestURL string      `json:"request_url"`
	FinalURL   string      `json:"final_url"`
	StatusCode int         `json:"status_code"`
	Status     string      `json:"status"`
	Proto      string      `json:"proto"`
	Headers    http.Header `json:"headers"`
}

const httpLabelWidth = 9

func (r *HTTPRecords) Render() string {
	lines := []string{
		field("Status", statusColor(r.StatusCode, r.Status)),
		field("Protocol", style.White(r.Proto)),
	}

	if r.FinalURL != "" && r.FinalURL != r.RequestURL {
		redirect := style.White(r.RequestURL) + style.Gray(" -> ") + style.White(r.FinalURL)
		lines = append(lines, field("Redirect", redirect))
	}

	lines = append(lines, label("Headers"))
	lines = append(lines, headerLines(r.Headers)...)

	return style.Section(r.Scheme, lines...)
}

func field(name, value string) string {
	return label(name) + " " + value
}

func label(name string) string {
	return "  " + style.Gray(fmt.Sprintf("%-*s:", httpLabelWidth, name))
}

func headerLines(headers http.Header) []string {
	if len(headers) == 0 {
		return []string{"    " + style.Dim("(none)")}
	}

	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		value := style.White(strings.Join(headers[k], ", "))
		lines = append(lines, "    "+style.Gray(k+":")+" "+value)
	}
	return lines
}

// statusColor tints the status line by response class so a failure stands out
// without having to read the code.
func statusColor(code int, status string) string {
	switch {
	case code >= 200 && code < 300:
		return style.Green(status)
	case code >= 300 && code < 400:
		return style.Cyan(status)
	case code >= 400 && code < 500:
		return style.Yellow(status)
	case code >= 500:
		return style.Red(status)
	default:
		return style.White(status)
	}
}
