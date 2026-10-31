package terminal

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

const (
	// Mouse button and drag reporting in SGR form, plus bracketed paste.
	enableInput  = "\x1b[?1002h\x1b[?1006h\x1b[?2004h"
	disableInput = "\x1b[?2004l\x1b[?1006l\x1b[?1002l"

	// The screen wraps lines itself. With the terminal's own wrapping off, a
	// row that comes out wider than expected (a wide emoji, say) is clipped
	// instead of spilling into the next row or scrolling the whole screen.
	noWrap      = "\x1b[?7l"
	wrap        = "\x1b[?7h"
	resetRegion = "\x1b[r"

	hideCursor = "\x1b[?25l"
	showCursor = "\x1b[?25h"

	// Terminals that don't support synchronized output ignore these.
	syncStart = "\x1b[?2026h"
	syncEnd   = "\x1b[?2026l"

	scrollStep    = 3
	toastDuration = 1500 * time.Millisecond
)

type row struct {
	line, start, end int
}

type selection struct {
	anchor, head pos
	// A finished selection stays highlighted, but only an active drag moves
	// or copies it.
	dragging bool
}

// Screen owns the terminal while the shell runs, like vim does. Everything the
// program prints is captured into a scrollback buffer and drawn from there,
// which is what makes scrolling, selection, search, folding and the status bar
// possible.
type Screen struct {
	tty    io.Writer
	ttyMu  sync.Mutex
	closed bool
	keys   *keyBuffer

	mu            sync.Mutex
	buf           *buffer
	width, height int
	offset        int // rows scrolled up from the bottom
	rows          []row
	lastFrame     []string
	lastWidth     int
	lastCursor    string
	sel           *selection
	toast         string
	toastUntil    time.Time
	onInterrupt   func()
	status        status
	search        *outputSearch

	redraw chan struct{}
	done   chan struct{}

	stdout, stderr *os.File
	pipe           *os.File
	copied         chan struct{}
}

func newScreen(tty io.Writer, width, height int) *Screen {
	return &Screen{
		tty:    tty,
		keys:   newKeyBuffer(),
		buf:    newBuffer(),
		width:  width,
		height: height,
		redraw: make(chan struct{}, 1),
		done:   make(chan struct{}),
	}
}

// startScreen takes over the terminal: it switches to the alternate screen,
// turns on mouse and paste reporting, and points os.Stdout and os.Stderr at
// the screen's buffer.
func startScreen(in, tty *os.File) (*Screen, error) {
	width, height, err := term.GetSize(int(tty.Fd()))
	if err != nil {
		width, height = 80, 24
	}

	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	s := newScreen(tty, width, height)
	s.stdout, s.stderr = os.Stdout, os.Stderr
	s.pipe = w
	s.copied = make(chan struct{})
	os.Stdout, os.Stderr = w, w

	s.writeTTY(enterAltScreen + enableInput + noWrap)

	go s.capture(r)
	go s.readInput(in)
	go s.renderLoop()
	go s.watchSize(int(tty.Fd()))

	s.requestRedraw()
	return s, nil
}

func (s *Screen) Close() {
	os.Stdout, os.Stderr = s.stdout, s.stderr
	s.pipe.Close()

	// A module process that is still running keeps the pipe open, so don't
	// wait for the last of the output forever.
	select {
	case <-s.copied:
	case <-time.After(500 * time.Millisecond):
	}

	close(s.done)

	s.ttyMu.Lock()
	defer s.ttyMu.Unlock()
	io.WriteString(s.tty, resetRegion+wrap+disableInput+showCursor+leaveAltScreen)
	s.closed = true
}

func (s *Screen) Input() io.Reader {
	return s.keys
}

func (s *Screen) CloseInput() {
	s.keys.closeWithError(io.EOF)
}

func (s *Screen) OnInterrupt(fn func()) (stop func()) {
	s.mu.Lock()
	s.onInterrupt = fn
	s.mu.Unlock()

	return func() {
		s.mu.Lock()
		s.onInterrupt = nil
		s.mu.Unlock()
	}
}

func (s *Screen) writeTTY(str string) {
	s.ttyMu.Lock()
	defer s.ttyMu.Unlock()

	// After Close, a late frame would land on the normal screen.
	if !s.closed {
		io.WriteString(s.tty, str)
	}
}

func (s *Screen) requestRedraw() {
	select {
	case s.redraw <- struct{}{}:
	default:
	}
}

func (s *Screen) capture(r *os.File) {
	defer close(s.copied)

	chunk := make([]byte, 32*1024)
	for {
		n, err := r.Read(chunk)
		if n > 0 {
			s.write(chunk[:n])
		}
		if err != nil {
			r.Close()
			return
		}
	}
}

func (s *Screen) write(p []byte) {
	s.mu.Lock()
	added := s.buf.write(p)
	// Keep the view still while scrolled up, instead of drifting with output.
	if s.offset > 0 {
		s.offset += added
	}
	s.mu.Unlock()
	s.requestRedraw()
}

func (s *Screen) renderLoop() {
	for {
		select {
		case <-s.done:
			return
		case <-s.redraw:
			s.render()
			// Fast output only needs a frame every few milliseconds.
			time.Sleep(8 * time.Millisecond)
		}
	}
}

func (s *Screen) watchSize(fd int) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			w, h, err := term.GetSize(fd)
			if err != nil {
				continue
			}
			s.mu.Lock()
			changed := w != s.width || h != s.height
			s.width, s.height = w, h
			s.mu.Unlock()
			if changed {
				s.requestRedraw()
			}
		}
	}
}

func (s *Screen) contentHeight() int {
	return max(1, s.height-1)
}

func (s *Screen) rowCount(i, w int) int {
	b := s.buf
	n := len(b.lines[i])
	if i == len(b.lines)-1 {
		n = max(n, b.col+1)
	}
	return max(1, (n+w-1)/w)
}

func (s *Screen) layout() (rows []row, cursorRow, cursorCol int) {
	w, h := max(s.width, 1), s.contentHeight()
	b := s.buf
	need := s.offset + h
	folded := s.foldedRanges()

	var rev []row
	cursorRev := -1
	for i := len(b.lines) - 1; i >= 0 && len(rev) < need; i-- {
		abs := b.dropped + i
		if inRanges(folded, abs) {
			continue
		}
		count := s.rowCount(i, w)
		if i == len(b.lines)-1 {
			cursorRev = count - 1 - b.col/w
		}
		for k := count - 1; k >= 0; k-- {
			start := k * w
			end := min(start+w, len(b.lines[i]))
			rev = append(rev, row{line: abs, start: start, end: max(start, end)})
		}
	}

	if s.offset > 0 && len(rev) < need {
		s.offset = max(0, len(rev)-h)
	}

	visible := rev[min(s.offset, len(rev)):min(len(rev), s.offset+h)]
	rows = make([]row, len(visible))
	for i, r := range visible {
		rows[len(visible)-1-i] = r
	}

	cursorRow = -1
	if s.offset == 0 && cursorRev >= 0 && cursorRev < len(visible) {
		cursorRow = len(visible) - 1 - cursorRev
		cursorCol = b.col % w
	}
	return rows, cursorRow, cursorCol
}

// render draws the screen, rewriting only the rows that changed since the last
// frame, so typing touches one row instead of repainting everything.
func (s *Screen) render() {
	s.mu.Lock()
	rows, cursorRow, cursorCol := s.layout()
	s.rows = rows

	frame := make([]string, s.height)
	for y := range s.contentHeight() {
		if y >= len(frame) {
			break
		}
		var f strings.Builder
		width := 0
		if y < len(rows) {
			width = s.drawRow(&f, rows[y])
		}
		// Clearing after the text, not before, so the row never shows blank.
		// A full row needs no clearing, and clearing it would eat its last
		// character on some terminals.
		if width < s.width {
			f.WriteString("\x1b[K")
		}
		frame[y] = f.String()
	}

	statusCursor := -1
	if s.height > 1 {
		frame[s.height-1], statusCursor = s.statusBar()
	}

	if time.Now().Before(s.toastUntil) && s.height > 0 {
		toast := " " + s.toast + " "
		frame[0] += fmt.Sprintf("\x1b[1;%dH\x1b[0;1;30;42m%s\x1b[0m", max(1, s.width-len([]rune(toast))), toast)
	}

	resized := s.lastWidth != s.width || len(s.lastFrame) != len(frame)

	var changed strings.Builder
	for y, line := range frame {
		if resized || s.lastFrame[y] != line {
			fmt.Fprintf(&changed, "\x1b[%d;1H%s", y+1, line)
		}
	}
	// The status bar goes out with every frame, so nothing can leave it
	// displaced for long.
	if changed.Len() > 0 && s.height > 1 && !resized && s.lastFrame[s.height-1] == frame[s.height-1] {
		fmt.Fprintf(&changed, "\x1b[%d;1H%s", s.height, frame[s.height-1])
	}
	s.lastFrame, s.lastWidth = frame, s.width

	cursor := hideCursor
	switch {
	case statusCursor >= 0:
		cursor = fmt.Sprintf("\x1b[%d;%dH%s", s.height, statusCursor+1, showCursor)
	case cursorRow >= 0:
		cursor = fmt.Sprintf("\x1b[%d;%dH%s", cursorRow+1, cursorCol+1, showCursor)
	}

	var out strings.Builder
	switch {
	case changed.Len() > 0:
		// Synchronized output: the terminal shows the frame all at once.
		out.WriteString(syncStart + hideCursor)
		if resized {
			// Only the output rows can scroll; the status bar below them stays.
			fmt.Fprintf(&out, "\x1b[1;%dr\x1b[2J", s.contentHeight())
		}
		out.WriteString(changed.String() + cursor + syncEnd)
	case cursor != s.lastCursor:
		out.WriteString(cursor)
	}
	s.lastCursor = cursor
	s.mu.Unlock()

	if out.Len() > 0 {
		s.writeTTY(out.String())
	}
}

func (s *Screen) drawRow(f *strings.Builder, r row) int {
	line := s.buf.lines[r.line-s.buf.dropped]
	links := findLinks(line)
	hits := s.searchHits(r.line)

	current := ""
	link := -1
	for col := r.start; col < r.end; col++ {
		c := line[col]

		if l := linkAt(links, col); l != link {
			if link >= 0 {
				f.WriteString(linkEnd)
			}
			if l >= 0 {
				f.WriteString(linkStart(links[l].url))
			}
			link = l
		}

		sgr := c.sgr
		if link >= 0 {
			sgr += "\x1b[4m"
		}
		switch hitAt(hits, col) {
		case hitCurrent:
			sgr += "\x1b[30;43m"
		case hitOther:
			sgr += "\x1b[30;47m"
		}
		if s.selected(pos{r.line, col}) {
			sgr += "\x1b[7m"
		}
		if sgr != current {
			f.WriteString("\x1b[0m" + sgr)
			current = sgr
		}
		f.WriteRune(c.r)
	}
	if link >= 0 {
		f.WriteString(linkEnd)
	}
	f.WriteString("\x1b[0m")

	width := r.end - r.start
	if r.end == len(line) {
		if n := s.foldedCount(r.line); n > 0 {
			note := fmt.Sprintf("  ▸ %d lines folded, click to expand", n)
			if width+len([]rune(note)) <= s.width {
				f.WriteString("\x1b[2m" + note + "\x1b[0m")
				width += len([]rune(note))
			}
		}
	}
	return width
}

func (s *Screen) selected(p pos) bool {
	if s.sel == nil {
		return false
	}
	a, z := s.sel.anchor, s.sel.head
	if z.before(a) {
		a, z = z, a
	}
	return !p.before(a) && !z.before(p)
}

func (s *Screen) at(x, y int) (pos, bool) {
	if len(s.rows) == 0 {
		return pos{}, false
	}
	r := s.rows[min(max(y-1, 0), len(s.rows)-1)]
	return pos{line: r.line, col: min(r.start+max(x-1, 0), max(r.end-1, r.start))}, true
}

func (s *Screen) lineText(abs int) []rune {
	i := abs - s.buf.dropped
	if i < 0 || i >= len(s.buf.lines) {
		return nil
	}
	line := s.buf.lines[i]
	text := make([]rune, len(line))
	for j, c := range line {
		text[j] = c.r
	}
	return text
}

func (s *Screen) showToast(msg string) {
	s.mu.Lock()
	s.toast = msg
	s.toastUntil = time.Now().Add(toastDuration)
	s.mu.Unlock()

	s.requestRedraw()
	time.AfterFunc(toastDuration, s.requestRedraw)
}

func (s *Screen) scroll(delta int) {
	s.mu.Lock()
	s.offset = max(0, s.offset+delta)
	s.mu.Unlock()
	s.requestRedraw()
}

func (s *Screen) scrollTo(abs, col int) {
	w := max(s.width, 1)
	folded := s.foldedRanges()
	below := 0
	for i := len(s.buf.lines) - 1; i > abs-s.buf.dropped; i-- {
		if !inRanges(folded, s.buf.dropped+i) {
			below += s.rowCount(i, w)
		}
	}
	if i := abs - s.buf.dropped; i >= 0 && i < len(s.buf.lines) {
		below += s.rowCount(i, w) - 1 - col/w
	}
	s.offset = max(0, below-s.contentHeight()/2)
}
