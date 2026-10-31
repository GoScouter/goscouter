package terminal

import (
	"strings"
)

type outputSearch struct {
	query   []rune
	matches []match
	current int
}

type match struct {
	line, col, n int
}

type hit int

const (
	noHit hit = iota
	hitOther
	hitCurrent
)

func (s *Screen) openSearch() {
	s.mu.Lock()
	s.search = &outputSearch{}
	s.sel = nil
	s.mu.Unlock()
	s.requestRedraw()
}

func (s *Screen) closeSearch() {
	s.mu.Lock()
	s.search = nil
	s.offset = 0
	s.mu.Unlock()
	s.requestRedraw()
}

func (s *Screen) searchKey(k string) {
	switch k {
	case string(rune(CtrlG)), string(rune(CtrlC)):
		s.closeSearch()
		return
	}

	s.mu.Lock()
	q := s.search
	if q == nil {
		s.mu.Unlock()
		return
	}

	switch k {
	case "\r", "\n", string(rune(CtrlF)), "\x1b[A":
		s.step(-1)
	case "\x1b[B":
		s.step(1)
	case string(rune(Backspace)), string(rune(CtrlH)):
		if len(q.query) > 0 {
			q.query = q.query[:len(q.query)-1]
			s.find()
		}
	default:
		if strings.HasPrefix(k, "\x1b") {
			break
		}
		for _, r := range k {
			if r >= 0x20 && r != Backspace {
				q.query = append(q.query, r)
			}
		}
		s.find()
	}
	s.mu.Unlock()
	s.requestRedraw()
}

func (s *Screen) find() {
	q := s.search
	q.matches = nil
	if len(q.query) == 0 {
		s.offset = 0
		return
	}

	needle := []rune(strings.ToLower(string(q.query)))
	for i := range s.buf.lines {
		abs := s.buf.dropped + i
		text := []rune(strings.ToLower(string(s.lineText(abs))))
		for col := 0; col+len(needle) <= len(text); col++ {
			if string(text[col:col+len(needle)]) == string(needle) {
				q.matches = append(q.matches, match{line: abs, col: col, n: len(needle)})
				col += len(needle) - 1
			}
		}
	}

	q.current = len(q.matches) - 1
	s.show()
}

func (s *Screen) step(delta int) {
	q := s.search
	if len(q.matches) == 0 {
		return
	}
	q.current = min(max(q.current+delta, 0), len(q.matches)-1)
	s.show()
}

func (s *Screen) show() {
	q := s.search
	if q.current < 0 || q.current >= len(q.matches) {
		return
	}
	m := q.matches[q.current]
	s.unfoldAround(m.line)
	s.scrollTo(m.line, m.col)
}

func (s *Screen) searchHits(line int) []match {
	q := s.search
	if q == nil {
		return nil
	}
	var hits []match
	for i, m := range q.matches {
		if m.line == line {
			if i == q.current {
				m.n = -m.n // marks the current match
			}
			hits = append(hits, m)
		}
	}
	return hits
}

func hitAt(hits []match, col int) hit {
	for _, m := range hits {
		n, kind := m.n, hitOther
		if n < 0 {
			n, kind = -n, hitCurrent
		}
		if col >= m.col && col < m.col+n {
			return kind
		}
	}
	return noHit
}
