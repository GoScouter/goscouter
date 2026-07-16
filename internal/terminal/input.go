package terminal

import (
	"bufio"
	"errors"
	"io"
)

const (
	CtrlC     = 0x03 // ETX  (Ctrl-C)
	CtrlD     = 0x04 // EOT  (Ctrl-D)
	Backspace = 0x7f // DEL
	CtrlH     = 0x08 // BS   (some terminals send this for backspace)
	Escape    = 0x1b // ESC  (start of an ANSI escape sequence)
	Up        = 'A'  // final byte of ESC [ A (up arrow)
	Down      = 'B'  // final byte of ESC [ B (down arrow)
    Carriage  = '\r'
	Newline   = '\n'
)

// User presses Ctrl-C.
var ErrInterrupted = errors.New("interrupted")

func ReadLine(in *bufio.Reader, out io.Writer, state *ShellState) (string, error) {
	var line []rune

	for {
		r, _, err := in.ReadRune()
		if err != nil {
			return string(line), err
		}

		switch r {
		case CtrlC:
			io.WriteString(out, "\r\n")
			return "", ErrInterrupted
		case CtrlD:
			if len(line) == 0 {
				return "", io.EOF
			}
		case Carriage, Newline:
			io.WriteString(out, "\r\n")
			return string(line), nil
		case Backspace, CtrlH:
			if len(line) > 0 {
				line = line[:len(line)-1]
				io.WriteString(out, "\b \b")
			}
		case Escape:
			// Arrow keys arrive as ESC [ <final>. Consume the '[' and the
			// final byte, then dispatch to history navigation.
			next, _, err := in.ReadRune()
			if err != nil {
				return string(line), err
			}
			// CSI ('[') is the common form; SS3 ('O') appears when the
			// terminal is in application cursor-keys mode.
			if next != '[' && next != 'O' {
				break
			}
			dir, _, err := in.ReadRune()
			if err != nil {
				return string(line), err
			}
			if dir != Up && dir != Down {
				break
			}

			cmd := state.Move(dir)
			if cmd == "" {
				break
			}

            for i := 0; i < len(line); i++ {
				io.WriteString(out, "\b \b")
			}
			line = []rune(cmd)
			io.WriteString(out, cmd)
        default:
			line = append(line, r)
			io.WriteString(out, string(r))
		}
	}
}
