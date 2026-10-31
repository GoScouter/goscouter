package terminal

import (
	"fmt"
	"strings"
	"time"
)

type status struct {
	target  string
	hint    string
	busy    string
	started time.Time
	stop    chan struct{}
}

const (
	barBase    = "\x1b[0;48;5;236;38;5;250m"
	barBadge   = "\x1b[1;38;5;141m"
	barDim     = "\x1b[38;5;245m"
	barAccent  = "\x1b[38;5;214m"
	barSearch  = "\x1b[1;38;5;255m"
	idleTips   = "Tab complete  ·  Ctrl-R history  ·  Ctrl-F search  ·  help"
	busyTips   = "Ctrl-C cancel"
	searchTips = "Enter/↑ older  ·  ↓ newer  ·  Esc close"
)

var spinner = []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")

func (s *Screen) SetTarget(target string) {
	s.mu.Lock()
	s.status.target = target
	s.mu.Unlock()
	s.requestRedraw()
}

func (s *Screen) SetHint(hint string) {
	s.mu.Lock()
	changed := s.status.hint != hint
	s.status.hint = hint
	s.mu.Unlock()
	if changed {
		s.requestRedraw()
	}
}

func (s *Screen) Busy(command string) {
	s.mu.Lock()
	if s.status.stop != nil {
		close(s.status.stop)
	}
	stop := make(chan struct{})
	s.status.busy, s.status.started, s.status.stop = command, time.Now(), stop
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-s.done:
				return
			case <-ticker.C:
				s.requestRedraw()
			}
		}
	}()
	s.requestRedraw()
}

func (s *Screen) Idle() {
	s.mu.Lock()
	if s.status.stop != nil {
		close(s.status.stop)
		s.status.stop = nil
	}
	s.status.busy = ""
	s.mu.Unlock()
	s.requestRedraw()
}

type segment struct {
	text, sgr string
}

func (s *Screen) statusBar() (string, int) {
	var left, right []segment
	cursor := -1

	if q := s.search; q != nil {
		count := "no matches"
		if len(q.matches) > 0 {
			count = fmt.Sprintf("%d/%d", q.current+1, len(q.matches))
		}
		left = []segment{
			{" search ", barBadge},
			{string(q.query), barSearch},
		}
		cursor = segmentsWidth(left)
		left = append(left, segment{"  " + count, barDim})
		right = []segment{{searchTips + " ", barDim}}
	} else {
		left = []segment{{" gs ", barBadge}}
		if s.status.target != "" {
			limit := min(maxTargetWidth, max(s.width/3, 8))
			left = append(left, segment{shortenTarget(s.status.target, limit) + "  ", ""})
		}

		if s.status.busy != "" {
			elapsed := time.Since(s.status.started)
			frame := spinner[int(elapsed/(100*time.Millisecond))%len(spinner)]
			left = append(left, segment{fmt.Sprintf("%c %s  %s", frame, s.status.busy, formatElapsed(elapsed)), barAccent})
			right = []segment{{busyTips + " ", barDim}}
		} else {
			tips := idleTips
			if s.status.hint != "" {
				tips = s.status.hint
			}
			left = append(left, segment{tips, barDim})
		}

		if s.offset > 0 {
			right = []segment{{"↓ scrolled up, type to return ", barAccent}}
		}
	}

	return bar(s.width, left, right), cursor
}

const maxTargetWidth = 32

func shorten(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n < 2 {
		return string(r[:n])
	}
	head := (n - 1) / 2
	tail := n - 1 - head
	return string(r[:head]) + "…" + string(r[len(r)-tail:])
}

func shortenTarget(target string, n int) string {
	if len([]rune(target)) <= n {
		return target
	}

	host := strings.TrimSuffix(target, "/")
	if _, after, ok := strings.Cut(host, "://"); ok {
		host = after
	}
	if len([]rune(host)) <= n {
		return host
	}
	if strings.Contains(host, "/") {
		// A path after the host: the ending isn't at the end to keep.
		return shorten(host, n)
	}

	root := domainRoot(host)
	if root == "" {
		return shorten(host, n)
	}

	rest := strings.TrimSuffix(host, root)
	room := n - len([]rune(root))
	if room < 2 {
		return "…" + root
	}
	return shorten(rest, room) + root
}

// secondLevel are the labels that form a domain ending together with the
// country after them, like "co" in ".co.il" or "com" in ".com.au".
var secondLevel = map[string]bool{
	"co": true, "com": true, "org": true, "net": true, "gov": true,
	"ac": true, "edu": true, "mil": true, "ltd": true, "plc": true,
}

func domainRoot(host string) string {
	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return ""
	}
	last := labels[len(labels)-1]
	if last == "" {
		return ""
	}
	if len(labels) >= 3 && len(last) == 2 && secondLevel[labels[len(labels)-2]] {
		return "." + labels[len(labels)-2] + "." + last
	}
	return "." + last
}

func formatElapsed(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
}

func segmentsWidth(segs []segment) int {
	n := 0
	for _, seg := range segs {
		n += len([]rune(seg.text))
	}
	return n
}

func bar(width int, left, right []segment) string {
	if segmentsWidth(left)+segmentsWidth(right) > width {
		right = nil
	}

	var b strings.Builder
	// Clearing the row in the bar's color first paints it edge to edge, even
	// if the terminal is wider than it reported.
	b.WriteString(barBase + "\x1b[K")
	used := 0
	write := func(segs []segment) {
		for _, seg := range segs {
			text := []rune(seg.text)
			if room := width - used; len(text) > room {
				text = text[:max(room, 0)]
			}
			b.WriteString(barBase + seg.sgr + string(text))
			used += len(text)
		}
	}

	write(left)
	if pad := width - used - segmentsWidth(right); pad > 0 {
		b.WriteString(barBase + strings.Repeat(" ", pad))
		used += pad
	}
	write(right)
	b.WriteString("\x1b[0m")
	return b.String()
}
