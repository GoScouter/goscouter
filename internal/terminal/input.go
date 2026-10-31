package terminal

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/GoScouter/sdk/style"
)

const (
	CtrlA     = 0x01 // move to start of line
	CtrlB     = 0x02 // move left
	CtrlC     = 0x03 // ETX, abandon the line
	CtrlD     = 0x04 // EOT, exit on an empty line, otherwise delete forward
	CtrlE     = 0x05 // move to end of line
	CtrlF     = 0x06 // move right (the screen takes it for searching the output)
	CtrlG     = 0x07 // cancel the history search; the screen sends it for Esc
	CtrlH     = 0x08 // BS, some terminals send this for backspace
	Tab       = 0x09 // complete the command or argument
	Newline   = '\n'
	CtrlK     = 0x0b // delete to end of line
	CtrlL     = 0x0c // clear the screen
	Carriage  = '\r'
	CtrlN     = 0x0e // next history entry
	CtrlP     = 0x10 // previous history entry
	CtrlR     = 0x12 // search the history
	CtrlU     = 0x15 // delete to start of line
	CtrlW     = 0x17 // delete the previous word
	Escape    = 0x1b // start of an ANSI escape sequence
	Backspace = 0x7f // DEL
)

const (
	clearToEnd  = "\x1b[K"
	clearScreen = "\x1b[H\x1b[2J"
)

// User presses Ctrl-C.
var ErrInterrupted = errors.New("interrupted")

type Completer func(line string) []string

type LineOptions struct {
	Prompt   string
	Complete Completer
	Hint     func(line string) string
}

type editor struct {
	in     *bufio.Reader
	out    io.Writer
	opts   LineOptions
	screen *Screen

	buf []rune
	pos int

	history []string
	histIdx int
	draft   []rune

	search *historySearch
}

type historySearch struct {
	query  []rune
	match  int // index into history, len(history) before the first match
	failed bool
	saved  []rune
}

func ReadLine(in *bufio.Reader, out io.Writer, state *ShellState, opts LineOptions) (string, error) {
	e := &editor{in: in, out: out, opts: opts}
	if state != nil {
		e.history = state.CommandHistory
		e.screen = state.Screen
	}
	e.histIdx = len(e.history)
	defer e.hint("")

	e.refresh(true)
	for {
		r, _, err := in.ReadRune()
		if err != nil {
			return string(e.buf), err
		}

		if e.search != nil && e.searchKey(r) {
			e.refresh(true)
			continue
		}

		switch r {
		case Carriage, Newline:
			e.refresh(false)
			io.WriteString(out, "\r\n")
			return string(e.buf), nil
		case CtrlC:
			e.refresh(false)
			io.WriteString(out, "^C\r\n")
			return "", ErrInterrupted
		case CtrlD:
			if len(e.buf) == 0 {
				io.WriteString(out, "\r\n")
				return "", io.EOF
			}
			e.deleteAt(e.pos)
		case Backspace, CtrlH:
			if e.pos > 0 {
				e.pos--
				e.deleteAt(e.pos)
			}
		case Tab:
			e.tab()
		case CtrlA:
			e.pos = 0
		case CtrlE:
			e.pos = len(e.buf)
		case CtrlB:
			e.left()
		case CtrlF:
			e.right()
		case CtrlK:
			e.buf = e.buf[:e.pos]
		case CtrlU:
			e.buf = slices.Delete(e.buf, 0, e.pos)
			e.pos = 0
		case CtrlW:
			e.deleteWord()
		case CtrlL:
			io.WriteString(out, clearScreen)
		case CtrlP:
			e.historyUp()
		case CtrlN:
			e.historyDown()
		case CtrlR:
			e.startSearch()
		case Escape:
			if err := e.escape(); err != nil {
				return string(e.buf), err
			}
		default:
			if r < 0x20 {
				continue
			}
			e.buf = slices.Insert(e.buf, e.pos, r)
			e.pos++
		}

		e.refresh(true)
	}
}

// escape handles the sequences that arrow, Home, End and Delete keys send:
// ESC [ <params> <final> (CSI), or ESC O <final> (SS3, application cursor keys).
func (e *editor) escape() error {
	next, _, err := e.in.ReadRune()
	if err != nil {
		return err
	}

	var params []rune
	var final rune
	switch next {
	case '[':
		for {
			r, _, err := e.in.ReadRune()
			if err != nil {
				return err
			}
			if r >= 0x40 && r <= 0x7e {
				final = r
				break
			}
			params = append(params, r)
		}
	case 'O':
		if final, _, err = e.in.ReadRune(); err != nil {
			return err
		}
	default:
		return nil
	}

	switch final {
	case 'A':
		e.historyUp()
	case 'B':
		e.historyDown()
	case 'C':
		e.right()
	case 'D':
		e.left()
	case 'H':
		e.pos = 0
	case 'F':
		e.pos = len(e.buf)
	case '~':
		switch string(params) {
		case "1", "7":
			e.pos = 0
		case "4", "8":
			e.pos = len(e.buf)
		case "3":
			e.deleteAt(e.pos)
		}
	}
	return nil
}

func (e *editor) left() {
	if e.pos > 0 {
		e.pos--
	}
}

func (e *editor) right() {
	if e.pos < len(e.buf) {
		e.pos++
		return
	}
	if cands := e.candidates(); len(cands) > 0 && e.ghost() != "" {
		e.setWord(cands[0])
	}
}

func (e *editor) deleteAt(i int) {
	if i < len(e.buf) {
		e.buf = slices.Delete(e.buf, i, i+1)
	}
}

func (e *editor) deleteWord() {
	i := e.pos
	for i > 0 && e.buf[i-1] == ' ' {
		i--
	}
	for i > 0 && e.buf[i-1] != ' ' {
		i--
	}
	e.buf = slices.Delete(e.buf, i, e.pos)
	e.pos = i
}

func (e *editor) set(s string) {
	e.buf = []rune(s)
	e.pos = len(e.buf)
}

func (e *editor) historyUp() {
	if e.histIdx == 0 {
		return
	}
	if e.histIdx == len(e.history) {
		e.draft = slices.Clone(e.buf)
	}
	e.histIdx--
	e.set(e.history[e.histIdx])
}

func (e *editor) historyDown() {
	if e.histIdx == len(e.history) {
		return
	}
	e.histIdx++
	if e.histIdx == len(e.history) {
		e.set(string(e.draft))
		return
	}
	e.set(e.history[e.histIdx])
}

func (e *editor) startSearch() {
	if e.search == nil {
		e.search = &historySearch{match: len(e.history), saved: slices.Clone(e.buf)}
	}
	e.findOlder(e.search.match - 1)
}

func (e *editor) findOlder(from int) {
	s := e.search
	query := strings.ToLower(string(s.query))
	for i := min(from, len(e.history)-1); i >= 0; i-- {
		if strings.Contains(strings.ToLower(e.history[i]), query) {
			s.match, s.failed = i, false
			e.set(e.history[i])
			return
		}
	}
	s.failed = true
}

func (e *editor) searchKey(r rune) bool {
	s := e.search
	switch {
	case r == CtrlR:
		e.findOlder(s.match - 1)
	case r == CtrlG || r == CtrlC:
		e.set(string(s.saved))
		e.search = nil
	case r == Backspace || r == CtrlH:
		if len(s.query) > 0 {
			s.query = s.query[:len(s.query)-1]
			e.findOlder(len(e.history) - 1)
		}
	case r >= 0x20 && r != Backspace:
		s.query = append(s.query, r)
		e.findOlder(s.match)
	default:
		e.search = nil
		return false
	}
	return true
}

func (e *editor) word() string {
	line := string(e.buf)
	return line[strings.LastIndexByte(line, ' ')+1:]
}

func (e *editor) setWord(s string) {
	line := string(e.buf)
	e.set(line[:strings.LastIndexByte(line, ' ')+1] + s)
}

func (e *editor) candidates() []string {
	if e.opts.Complete == nil || e.pos != len(e.buf) || e.search != nil {
		return nil
	}
	line := string(e.buf)
	if strings.HasPrefix(line, "!") {
		return nil
	}
	return e.opts.Complete(line)
}

func (e *editor) ghost() string {
	word := e.word()
	if word == "" {
		return ""
	}
	cands := e.candidates()
	if len(cands) == 0 {
		return ""
	}
	return remainder(word, cands[0])
}

// remainder returns what is left of cand after word, matching either the full
// name or the part after the namespace ("subdo" -> "mains" for
// "internal:subdomains").
func remainder(word, cand string) string {
	if rest, ok := strings.CutPrefix(cand, word); ok {
		return rest
	}
	if _, short, ok := strings.Cut(cand, ":"); ok {
		if rest, ok := strings.CutPrefix(short, word); ok {
			return rest
		}
	}
	return ""
}

func (e *editor) tab() {
	cands := e.candidates()
	switch {
	case len(cands) == 0:
		return
	case len(cands) == 1:
		e.setWord(cands[0] + " ")
		return
	}

	word := e.word()
	if prefix := commonPrefix(cands); len(prefix) > len(word) && strings.HasPrefix(prefix, word) {
		e.setWord(prefix)
		return
	}

	var b strings.Builder
	b.WriteString(clearToEnd + "\r\n")
	for i, c := range cands {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(style.Cyan(c))
	}
	b.WriteString("\r\n")
	io.WriteString(e.out, b.String())
}

func commonPrefix(words []string) string {
	prefix := words[0]
	for _, w := range words[1:] {
		for !strings.HasPrefix(w, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}

func (e *editor) hint(h string) {
	if e.screen != nil {
		e.screen.SetHint(h)
	}
}

func (e *editor) refresh(withGhost bool) {
	var b strings.Builder
	b.WriteString("\r")

	if s := e.search; s != nil {
		label := "history search"
		if s.failed {
			label = "no match in history"
		}
		b.WriteString(style.Dim(label+" ") + style.Cyan(string(s.query)) + style.Dim(": "))
		e.hint("Enter run  ·  → edit  ·  Ctrl-R older  ·  Esc cancel")
	} else {
		b.WriteString(e.opts.Prompt)
		if e.opts.Hint != nil {
			e.hint(e.opts.Hint(string(e.buf)))
		}
	}
	b.WriteString(string(e.buf))

	back := len(e.buf) - e.pos
	if withGhost {
		if ghost := e.ghost(); ghost != "" {
			b.WriteString(style.Gray(ghost))
			back += len([]rune(ghost))
		}
	}

	b.WriteString(clearToEnd)
	if back > 0 {
		fmt.Fprintf(&b, "\x1b[%dD", back)
	}
	io.WriteString(e.out, b.String())
}
