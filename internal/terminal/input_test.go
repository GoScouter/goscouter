package terminal

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"testing"
)

func readLine(input string) (string, string, error) {
	in := bufio.NewReader(strings.NewReader(input))
	var out strings.Builder
	line, err := ReadLine(in, &out)
	return line, out.String(), err
}

func TestReadLineCarriageReturn(t *testing.T) {
	line, _, err := readLine("hello\r")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line != "hello" {
		t.Fatalf("expected %q, got %q", "hello", line)
	}
}

func TestReadLineNewline(t *testing.T) {
	line, _, err := readLine("world\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line != "world" {
		t.Fatalf("expected %q, got %q", "world", line)
	}
}

func TestReadLineCtrlC(t *testing.T) {
	line, _, err := readLine("abc\x03")
	if !errors.Is(err, ErrInterrupted) {
		t.Fatalf("expected ErrInterrupted, got %v", err)
	}
	if line != "" {
		t.Fatalf("expected empty line on interrupt, got %q", line)
	}
}

func TestReadLineCtrlDEmptyLine(t *testing.T) {
	line, _, err := readLine("\x04")
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF, got %v", err)
	}
	if line != "" {
		t.Fatalf("expected empty line, got %q", line)
	}
}

func TestReadLineCtrlDIgnoredWithContent(t *testing.T) {
	// Ctrl-D with a non-empty buffer is ignored; the following CR ends the line.
	line, _, err := readLine("hi\x04\r")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line != "hi" {
		t.Fatalf("expected %q, got %q", "hi", line)
	}
}

func TestReadLineBackspace(t *testing.T) {
	// "ab", backspace, "c" -> "ac"
	line, _, err := readLine("ab\x7fc\r")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line != "ac" {
		t.Fatalf("expected %q, got %q", "ac", line)
	}
}

func TestReadLineBackspaceOnEmptyLine(t *testing.T) {
	line, out, err := readLine("\x7f\r")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line != "" {
		t.Fatalf("expected empty line, got %q", line)
	}
	// Nothing to erase, so no backspace sequence should be emitted.
	if strings.Contains(out, "\b") {
		t.Fatalf("expected no backspace output, got %q", out)
	}
}

func TestReadLineIgnoresControlChars(t *testing.T) {
	// A bare tab (0x09) is an unhandled control char and should be dropped.
	line, _, err := readLine("a\tb\r")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line != "ab" {
		t.Fatalf("expected %q, got %q", "ab", line)
	}
}

func TestReadLineEchoesInput(t *testing.T) {
	_, out, err := readLine("xy\r")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(out, "xy") {
		t.Fatalf("expected echoed input to start with %q, got %q", "xy", out)
	}
}

func TestReadLineErrorPropagates(t *testing.T) {
	// No terminator: the underlying reader returns io.EOF, which surfaces.
	line, _, err := readLine("partial")
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF, got %v", err)
	}
	if line != "partial" {
		t.Fatalf("expected buffered %q returned with error, got %q", "partial", line)
	}
}
