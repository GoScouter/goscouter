package terminal

import (
	"bufio"
	"encoding/base64"
	"io"
	"strconv"
	"strings"
)

// readInput splits terminal input into what the screen handles itself (mouse
// events, pastes, output search) and everything else, which goes to the line
// editor.
func (s *Screen) readInput(in io.Reader) {
	r := bufio.NewReader(in)
	for {
		if err := s.readEvent(r); err != nil {
			s.keys.closeWithError(err)
			return
		}
	}
}

func (s *Screen) readEvent(r *bufio.Reader) error {
	c, _, err := r.ReadRune()
	if err != nil {
		return err
	}

	if c != Escape {
		s.handleKey(string(c))
		return nil
	}

	// A key's escape sequence arrives in one piece, so an ESC with nothing
	// after it is the Esc key itself.
	if r.Buffered() == 0 {
		s.handleKey(string(rune(CtrlG)))
		return nil
	}

	next, _, err := r.ReadRune()
	if err != nil {
		return err
	}
	if next != '[' {
		s.handleKey(string([]rune{c, next}))
		return nil
	}

	var params []rune
	var final rune
	for {
		p, _, err := r.ReadRune()
		if err != nil {
			return err
		}
		if p >= 0x40 && p <= 0x7e {
			final = p
			break
		}
		params = append(params, p)
	}

	switch p := string(params); {
	case strings.HasPrefix(p, "<") && (final == 'M' || final == 'm'):
		s.mouse(p[1:], final == 'M')
	case p == "200" && final == '~':
		return s.paste(r)
	default:
		s.handleKey("\x1b[" + p + string(final))
	}
	return nil
}

func (s *Screen) handleKey(k string) {
	s.mu.Lock()
	searching := s.search != nil
	interrupt := s.onInterrupt
	s.mu.Unlock()

	switch {
	case searching:
		s.searchKey(k)
		return
	case k == string(rune(CtrlF)):
		s.openSearch()
		return
	case k == string(rune(CtrlC)) && interrupt != nil:
		interrupt()
		return
	}

	s.key(k)
}

func (s *Screen) key(k string) {
	s.mu.Lock()
	s.offset = 0
	s.sel = nil
	s.mu.Unlock()

	s.keys.write(k)
	s.requestRedraw()
}

func (s *Screen) mouse(params string, pressed bool) {
	fields := strings.Split(params, ";")
	if len(fields) != 3 {
		return
	}
	button, _ := strconv.Atoi(fields[0])
	x, _ := strconv.Atoi(fields[1])
	y, _ := strconv.Atoi(fields[2])

	const (
		ctrl   = 16
		motion = 32
		wheel  = 64
	)

	switch {
	case button&wheel != 0:
		if button&1 == 0 {
			s.scroll(scrollStep)
		} else {
			s.scroll(-scrollStep)
		}
		return
	case button&3 != 0:
		return
	}

	s.mu.Lock()
	if pressed && button&motion == 0 && y > s.contentHeight() {
		s.mu.Unlock()
		return
	}
	p, ok := s.at(x, y)
	if !ok {
		s.mu.Unlock()
		return
	}

	var copied, url string
	switch {
	case pressed && button&motion == 0 && button&ctrl != 0: // Ctrl+click
		url = urlAt(s.buf, p)
	case pressed && button&motion == 0: // press starts a new selection
		s.sel = &selection{anchor: p, head: p, dragging: true}
	case pressed && s.sel != nil && s.sel.dragging: // drag
		s.sel.head = p
	case !pressed && s.sel != nil && s.sel.dragging: // release
		s.sel.head = p
		s.sel.dragging = false
		if s.sel.anchor == s.sel.head {
			s.sel = nil
			s.toggleFold(p.line)
		} else {
			copied = s.buf.text(s.sel.anchor, s.sel.head)
		}
	}
	s.mu.Unlock()

	if copied != "" {
		s.copy(copied)
	}
	if url != "" {
		go openURL(url)
		s.showToast("↗ Opening link")
	}
	s.requestRedraw()
}

// copy puts text on the clipboard with OSC 52, which the terminal handles.
func (s *Screen) copy(text string) {
	s.writeTTY("\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(text)) + "\a")
	s.showToast("✓ Copied")
}

// paste reads a bracketed paste up to its end marker and types it into the
// line. Newlines and tabs become spaces so a paste never runs a command.
func (s *Screen) paste(r *bufio.Reader) error {
	const end = "\x1b[201~"

	var b strings.Builder
	for !strings.HasSuffix(b.String(), end) {
		c, _, err := r.ReadRune()
		if err != nil {
			return err
		}
		b.WriteRune(c)
	}

	text := strings.TrimSuffix(b.String(), end)
	text = strings.Map(func(r rune) rune {
		switch {
		case r == '\r' || r == '\n' || r == '\t':
			return ' '
		case r < 0x20 || r == 0x7f:
			return -1
		}
		return r
	}, text)
	text = strings.TrimSpace(text)

	if text == "" {
		return nil
	}

	s.mu.Lock()
	searching := s.search != nil
	s.mu.Unlock()
	if searching {
		s.searchKey(text)
	} else {
		s.key(text)
	}
	s.showToast("✓ Pasted")
	return nil
}
