package terminal

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

const maxLines = 5000

type cell struct {
	r   rune
	sgr string
}

type parserState int

const (
	ground parserState = iota
	escape
	csi
	osc
	oscEscape
)

// buffer is a minimal terminal emulator that turns everything the program
// prints into lines of cells. It understands what goscouter actually prints:
// text, colors, \r, \n, \b, clearing the line or screen and moving the cursor
// left and right on the last line. Anything else is dropped.
type buffer struct {
	lines   [][]cell
	dropped int // lines trimmed off the top, so positions stay stable
	col     int
	sgr     string
	blocks  []block

	state   parserState
	params  []byte
	pending []byte
}

func newBuffer() *buffer {
	return &buffer{lines: [][]cell{{}}}
}

func (b *buffer) write(p []byte) (newLines int) {
	data := p
	if len(b.pending) > 0 {
		data = append(b.pending, p...)
		b.pending = nil
	}

	for len(data) > 0 {
		if !utf8.FullRune(data) {
			b.pending = append([]byte(nil), data...)
			break
		}
		r, n := utf8.DecodeRune(data)
		data = data[n:]
		if b.feed(r) {
			newLines++
		}
	}
	return newLines
}

func (b *buffer) feed(r rune) (newLine bool) {
	switch b.state {
	case ground:
		switch {
		case r == Escape:
			b.state = escape
		case r == '\r':
			b.col = 0
		case r == '\n':
			b.newLine()
			return true
		case r == '\b':
			b.col = max(0, b.col-1)
		case r == '\t':
			for {
				b.put(' ')
				if b.col%8 == 0 {
					break
				}
			}
		case r >= 0x20 && r != 0x7f:
			b.put(r)
		}
	case escape:
		switch r {
		case '[':
			b.state = csi
			b.params = b.params[:0]
		case ']':
			b.state = osc
			b.params = b.params[:0]
		default:
			b.state = ground
		}
	case csi:
		if r >= 0x40 && r <= 0x7e {
			b.control(string(b.params), r)
			b.state = ground
		} else {
			b.params = utf8.AppendRune(b.params, r)
		}
	case osc:
		// Title changes and the like end with BEL or ESC \.
		switch r {
		case 0x07:
			b.endOSC()
		case Escape:
			b.state = oscEscape
		default:
			b.params = utf8.AppendRune(b.params, r)
		}
	case oscEscape:
		b.endOSC()
	}
	return false
}

func (b *buffer) last() *[]cell {
	return &b.lines[len(b.lines)-1]
}

func (b *buffer) put(r rune) {
	line := b.last()
	for len(*line) < b.col {
		*line = append(*line, cell{r: ' '})
	}
	c := cell{r: r, sgr: b.sgr}
	if b.col < len(*line) {
		(*line)[b.col] = c
	} else {
		*line = append(*line, c)
	}
	b.col++
}

func (b *buffer) newLine() {
	b.lines = append(b.lines, []cell{})
	b.col = 0
	if over := len(b.lines) - maxLines; over > 0 {
		b.lines = b.lines[over:]
		b.dropped += over
		for len(b.blocks) > 0 && b.blocks[0].header < b.dropped {
			b.blocks = b.blocks[1:]
		}
	}
}

// BlockMarker is printed when a command starts. It never reaches the terminal:
// the buffer takes it to mean that the line above, the prompt the command was
// typed on, starts a new block of output that can be folded.
const BlockMarker = "\x1b]" + blockOSC + "\x07"

const blockOSC = "1337;goscouter-block"

type block struct {
	header int
	folded bool
}

func (b *buffer) endOSC() {
	b.state = ground
	if string(b.params) == blockOSC {
		header := b.dropped + len(b.lines) - 1
		if b.col == 0 && len(b.lines[len(b.lines)-1]) == 0 {
			header-- // the usual case: the cursor is on the line after the prompt
		}
		b.blocks = append(b.blocks, block{header: max(header, b.dropped)})
	}
}

func (b *buffer) control(params string, final rune) {
	n := 1
	if v, err := strconv.Atoi(params); err == nil && v > 0 {
		n = v
	}

	switch final {
	case 'm':
		switch {
		case params == "" || params == "0":
			b.sgr = ""
		case strings.HasPrefix(params, "0;"):
			b.sgr = "\x1b[" + params + "m"
		default:
			b.sgr += "\x1b[" + params + "m"
		}
	case 'K':
		line := b.last()
		switch params {
		case "", "0":
			if b.col < len(*line) {
				*line = (*line)[:b.col]
			}
		case "2":
			*line = (*line)[:0]
		}
	case 'D':
		b.col = max(0, b.col-n)
	case 'C':
		b.col += n
	case 'G':
		b.col = n - 1
	case 'H':
		b.col = 0
	case 'J':
		if params == "2" || params == "3" {
			b.dropped += len(b.lines)
			b.lines = [][]cell{{}}
			b.col = 0
			b.blocks = nil
		}
	}
}

// pos is a place in the buffer. line counts from the very first line ever
// written, so it stays valid when old lines are trimmed.
type pos struct {
	line, col int
}

func (p pos) before(q pos) bool {
	return p.line < q.line || (p.line == q.line && p.col < q.col)
}

func (b *buffer) text(a, z pos) string {
	if z.before(a) {
		a, z = z, a
	}

	var out []string
	for abs := a.line; abs <= z.line; abs++ {
		i := abs - b.dropped
		if i < 0 || i >= len(b.lines) {
			continue
		}
		line := b.lines[i]

		start, end := 0, len(line)
		if abs == a.line {
			start = min(a.col, len(line))
		}
		if abs == z.line {
			end = min(z.col+1, len(line))
		}

		var s strings.Builder
		for _, c := range line[start:max(start, end)] {
			s.WriteRune(c.r)
		}
		out = append(out, strings.TrimRight(s.String(), " "))
	}
	return strings.Join(out, "\n")
}
