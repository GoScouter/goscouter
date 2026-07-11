package terminal

import (
    "bufio"
    "errors"
    "io"
)

const (
    ctrlC     = 0x03 // ETX  (Ctrl-C)
    ctrlD     = 0x04 // EOT  (Ctrl-D)
    backspace = 0x7f // DEL
    ctrlH     = 0x08 // BS   (some terminals send this for backspace)
    carriage  = '\r'
    newline   = '\n'
)

// User presses Ctrl-C.
var ErrInterrupted = errors.New("interrupted")

func ReadLine(in *bufio.Reader, out io.Writer) (string, error) {
    var line []rune

    for {
        r, _, err := in.ReadRune()
        if err != nil {
            return string(line), err
        }

        switch r {
        case ctrlC:
            io.WriteString(out, "\r\n")
            return "", ErrInterrupted
        case ctrlD:
            if len(line) == 0 {
                return "", io.EOF
            }
        case carriage, newline:
            io.WriteString(out, "\r\n")
            return string(line), nil
        case backspace, ctrlH:
            if len(line) > 0 {
                line = line[:len(line)-1]
                // Move cursor back, overwrite with space, move back again.
                io.WriteString(out, "\b \b")
            }

        default:
            if r < 0x20 {
                // Ignore other unhandled control characters.
                continue
            }
            line = append(line, r)
            io.WriteString(out, string(r))
        }
    }
}
