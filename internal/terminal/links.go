package terminal

import (
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"unicode/utf8"
)

var urlRE = regexp.MustCompile(`https?://[^\s"'<>` + "`" + `]+`)

type link struct {
	start, end int
	url        string
}

const linkEnd = "\x1b]8;;\x1b\\"

// linkStart opens an OSC 8 hyperlink, which terminals that support it make
// clickable.
func linkStart(url string) string {
	return "\x1b]8;;" + url + "\x1b\\"
}

func findLinks(line []cell) []link {
	if len(line) < len("http://x") {
		return nil
	}
	text := make([]rune, len(line))
	for i, c := range line {
		text[i] = c.r
	}
	str := string(text)

	var links []link
	for _, m := range urlRE.FindAllStringIndex(str, -1) {
		url := strings.TrimRight(str[m[0]:m[1]], ".,;:!?)]}")
		start := utf8.RuneCountInString(str[:m[0]])
		links = append(links, link{start: start, end: start + utf8.RuneCountInString(url), url: url})
	}
	return links
}

func linkAt(links []link, col int) int {
	for i, l := range links {
		if col >= l.start && col < l.end {
			return i
		}
	}
	return -1
}

func urlAt(b *buffer, p pos) string {
	i := p.line - b.dropped
	if i < 0 || i >= len(b.lines) {
		return ""
	}
	links := findLinks(b.lines[i])
	if l := linkAt(links, p.col); l >= 0 {
		return links[l].url
	}
	return ""
}

func openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		switch {
		case !isWSL():
			cmd = exec.Command("xdg-open", url)
		case hasCommand("wslview"):
			cmd = exec.Command("wslview", url)
		default:
			// WSL without wslu: let Windows open it.
			cmd = exec.Command("explorer.exe", url)
		}
	}

	if err := cmd.Start(); err == nil {
		go cmd.Wait()
	}
}

func isWSL() bool {
	version, err := os.ReadFile("/proc/version")
	return err == nil && strings.Contains(strings.ToLower(string(version)), "microsoft")
}

func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
