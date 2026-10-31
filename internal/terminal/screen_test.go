package terminal

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"testing"
)

func lineText(b *buffer, i int) string {
	var s strings.Builder
	for _, c := range b.lines[i] {
		s.WriteRune(c.r)
	}
	return s.String()
}

func bufferOf(input string) *buffer {
	b := newBuffer()
	b.write([]byte(input))
	return b
}

func TestBufferLines(t *testing.T) {
	b := bufferOf("one\r\ntwo\nthree")
	if len(b.lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(b.lines))
	}
	for i, want := range []string{"one", "two", "three"} {
		if got := lineText(b, i); got != want {
			t.Errorf("line %d = %q, want %q", i, got, want)
		}
	}
}

func TestBufferCarriageReturnOverwrites(t *testing.T) {
	if got := lineText(bufferOf("hello\rJ"), 0); got != "Jello" {
		t.Fatalf("got %q, want %q", got, "Jello")
	}
}

func TestBufferLineEditorRedraw(t *testing.T) {
	// What the line editor writes when redrawing "abc" with the cursor after "a".
	b := bufferOf("> abcdef\r> abc\x1b[K\x1b[2D")
	if got := lineText(b, 0); got != "> abc" {
		t.Fatalf("got %q, want %q", got, "> abc")
	}
	if b.col != 3 {
		t.Fatalf("cursor at %d, want 3", b.col)
	}
}

func TestBufferKeepsColors(t *testing.T) {
	b := bufferOf("\x1b[31mred\x1b[0m plain")
	if b.lines[0][0].sgr != "\x1b[31m" {
		t.Fatalf("expected red on the first cell, got %q", b.lines[0][0].sgr)
	}
	if b.lines[0][4].sgr != "" {
		t.Fatalf("expected no color after the reset, got %q", b.lines[0][4].sgr)
	}
}

func TestBufferClearScreen(t *testing.T) {
	b := bufferOf("old\nlines\x1b[H\x1b[2Jnew")
	if len(b.lines) != 1 || lineText(b, 0) != "new" {
		t.Fatalf("expected only %q after clearing, got %d lines", "new", len(b.lines))
	}
}

func TestBufferDropsOtherEscapes(t *testing.T) {
	if got := lineText(bufferOf("\x1b[?25la\x1b]0;title\x07b"), 0); got != "ab" {
		t.Fatalf("got %q, want %q", got, "ab")
	}
}

func TestBufferSplitUTF8(t *testing.T) {
	b := newBuffer()
	check := []byte("✓")
	b.write(check[:1])
	b.write(check[1:])
	if got := lineText(b, 0); got != "✓" {
		t.Fatalf("got %q, want %q", got, "✓")
	}
}

func TestBufferSplitEscape(t *testing.T) {
	b := newBuffer()
	b.write([]byte("a\x1b["))
	b.write([]byte("31mb"))
	if got := lineText(b, 0); got != "ab" {
		t.Fatalf("got %q, want %q", got, "ab")
	}
}

func TestBufferTrimsScrollback(t *testing.T) {
	b := newBuffer()
	for i := range maxLines + 10 {
		b.write(fmt.Appendf(nil, "%d\n", i))
	}
	if len(b.lines) != maxLines {
		t.Fatalf("expected %d lines, got %d", maxLines, len(b.lines))
	}
	if b.dropped != 11 {
		t.Fatalf("expected 11 dropped lines, got %d", b.dropped)
	}
}

func TestBufferText(t *testing.T) {
	b := bufferOf("first line\nsecond line\nthird")
	cases := []struct {
		a, z pos
		want string
	}{
		{pos{0, 0}, pos{0, 4}, "first"},
		{pos{0, 6}, pos{1, 5}, "line\nsecond"},
		{pos{1, 5}, pos{0, 6}, "line\nsecond"},
		{pos{2, 0}, pos{2, 99}, "third"},
	}
	for _, c := range cases {
		if got := b.text(c.a, c.z); got != c.want {
			t.Errorf("text(%v, %v) = %q, want %q", c.a, c.z, got, c.want)
		}
	}
}

type syncWriter struct {
	mu sync.Mutex
	b  strings.Builder
}

func (w *syncWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.Write(p)
}

func (w *syncWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.String()
}

// testScreen makes a screen with height rows of output, plus the status bar.
func testScreen(width, height int, output string) (*Screen, *syncWriter) {
	tty := &syncWriter{}
	s := newScreen(tty, width, height+1)
	s.write([]byte(output))
	s.render()
	return s, tty
}

func visible(s *Screen) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, _, _ := s.layout()
	out := make([]string, len(rows))
	for i, r := range rows {
		line := s.buf.lines[r.line-s.buf.dropped]
		var b strings.Builder
		for _, c := range line[r.start:r.end] {
			b.WriteRune(c.r)
		}
		out[i] = b.String()
	}
	return out
}

func numbered(n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	b.WriteString("> ")
	return b.String()
}

func TestLayoutShowsBottom(t *testing.T) {
	s, _ := testScreen(20, 3, numbered(10))
	got := strings.Join(visible(s), "|")
	if got != "line 9|line 10|> " {
		t.Fatalf("got %q", got)
	}
}

func TestLayoutWrapsLongLines(t *testing.T) {
	s, _ := testScreen(4, 5, "abcdefghij")
	got := strings.Join(visible(s), "|")
	if got != "abcd|efgh|ij" {
		t.Fatalf("got %q", got)
	}
}

func TestLayoutCursor(t *testing.T) {
	s, _ := testScreen(20, 5, "a\nb\n> ab\x1b[1D")
	s.mu.Lock()
	_, row, col := s.layout()
	s.mu.Unlock()
	if row != 2 || col != 3 {
		t.Fatalf("cursor at (%d, %d), want (2, 3)", row, col)
	}
}

func TestScrollUpAndBack(t *testing.T) {
	s, _ := testScreen(20, 3, numbered(10))

	s.scroll(2)
	if got := strings.Join(visible(s), "|"); got != "line 7|line 8|line 9" {
		t.Fatalf("after scrolling up got %q", got)
	}

	s.scroll(100)
	if got := visible(s)[0]; got != "line 1" {
		t.Fatalf("expected the top to be line 1, got %q", got)
	}

	s.key("x")
	if got := visible(s)[2]; got != "> " {
		t.Fatalf("expected the prompt at the bottom, got %q", got)
	}
}

func TestScrolledViewStaysStillOnOutput(t *testing.T) {
	s, _ := testScreen(20, 3, numbered(10))
	s.scroll(2)
	before := strings.Join(visible(s), "|")
	s.write([]byte("\nmore output\n"))
	if after := strings.Join(visible(s), "|"); after != before {
		t.Fatalf("view moved from %q to %q", before, after)
	}
}

func TestCursorHiddenWhileScrolled(t *testing.T) {
	s, _ := testScreen(20, 3, numbered(10))
	s.scroll(2)
	s.mu.Lock()
	_, row, _ := s.layout()
	s.mu.Unlock()
	if row != -1 {
		t.Fatalf("expected no cursor while scrolled up, got row %d", row)
	}
}

func events(s *Screen, input string) {
	r := bufio.NewReader(strings.NewReader(input))
	for s.readEvent(r) == nil {
	}
}

func keysRead(s *Screen) string {
	s.keys.closeWithError(io.EOF)
	b, _ := io.ReadAll(s.keys)
	return string(b)
}

func TestInputPassesKeysThrough(t *testing.T) {
	s, _ := testScreen(20, 3, "")
	events(s, "ab\x1b[A\x1bOB\x1b[3~\r")
	if got := keysRead(s); got != "ab\x1b[A\x1bOB\x1b[3~\r" {
		t.Fatalf("got %q", got)
	}
}

func TestInputWheelScrolls(t *testing.T) {
	s, _ := testScreen(20, 3, numbered(10))
	events(s, "\x1b[<64;5;2M")
	if s.offset != scrollStep {
		t.Fatalf("expected offset %d after wheel up, got %d", scrollStep, s.offset)
	}
	events(s, "\x1b[<65;5;2M")
	if s.offset != 0 {
		t.Fatalf("expected offset 0 after wheel down, got %d", s.offset)
	}
	if got := keysRead(s); got != "" {
		t.Fatalf("mouse events must not reach the editor, got %q", got)
	}
}

func TestInputDragCopies(t *testing.T) {
	s, tty := testScreen(20, 3, "hello world\nsecond\n> ")
	// Press on "w", drag, release on "d" (columns are 1-based).
	events(s, "\x1b[<0;7;1M\x1b[<32;9;1M\x1b[<0;11;1m")

	want := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte("world")) + "\a"
	if !strings.Contains(tty.String(), want) {
		t.Fatalf("expected %q to be copied, terminal got %q", "world", tty.String())
	}
	if s.toast != "✓ Copied" {
		t.Fatalf("expected a copied toast, got %q", s.toast)
	}
}

func TestInputClickDoesNotCopy(t *testing.T) {
	s, tty := testScreen(20, 3, "hello\n> ")
	events(s, "\x1b[<0;2;1M\x1b[<0;2;1m")
	if strings.Contains(tty.String(), "\x1b]52;") {
		t.Fatalf("a click must not copy, terminal got %q", tty.String())
	}
	if s.sel != nil {
		t.Fatal("a click must not leave a selection")
	}
}

func TestInputPaste(t *testing.T) {
	s, _ := testScreen(20, 3, "")
	events(s, "\x1b[200~line one\r\nline\ttwo\n\x1b[201~")
	if got := keysRead(s); got != "line one  line two" {
		t.Fatalf("got %q", got)
	}
	if s.toast != "✓ Pasted" {
		t.Fatalf("expected a pasted toast, got %q", s.toast)
	}
}

func TestInputInterrupt(t *testing.T) {
	s, _ := testScreen(20, 3, "")
	called := false
	stop := s.OnInterrupt(func() { called = true })
	events(s, "\x03")
	stop()
	events(s, "\x03")

	if !called {
		t.Fatal("expected Ctrl-C to call the interrupt handler")
	}
	if got := keysRead(s); got != "\x03" {
		t.Fatalf("expected only the second Ctrl-C to reach the editor, got %q", got)
	}
}

func TestRenderShowsToast(t *testing.T) {
	s, tty := testScreen(30, 3, "> ")
	s.showToast("✓ Copied")
	s.render()
	if !strings.Contains(tty.String(), "✓ Copied") {
		t.Fatalf("expected the toast in the frame, got %q", tty.String())
	}
}

func TestRenderOnlyRewritesChangedRows(t *testing.T) {
	s, tty := testScreen(20, 5, "line 1\nline 2\n> ab")
	tty.mu.Lock()
	tty.b.Reset()
	tty.mu.Unlock()

	// What the line editor sends after typing "c".
	s.write([]byte("\r> abc\x1b[K"))
	s.render()
	frame := tty.String()

	if !strings.Contains(frame, "\x1b[3;1H") {
		t.Fatalf("expected the prompt row to be redrawn, got %q", frame)
	}
	for _, row := range []string{"\x1b[1;1H", "\x1b[2;1H", "\x1b[4;1H", "\x1b[5;1H"} {
		if strings.Contains(frame, row) {
			t.Fatalf("expected only the prompt row to be redrawn, got %q", frame)
		}
	}
	if strings.Contains(frame, "\x1b[2J") {
		t.Fatalf("typing must not clear the screen, got %q", frame)
	}
}

func TestRenderNothingChanged(t *testing.T) {
	s, tty := testScreen(20, 5, "> ab")
	tty.mu.Lock()
	tty.b.Reset()
	tty.mu.Unlock()

	s.render()
	if got := tty.String(); got != "" {
		t.Fatalf("expected no output for an unchanged frame, got %q", got)
	}
}

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)

func bufioReader(r io.Reader) *bufio.Reader {
	return bufio.NewReader(r)
}
