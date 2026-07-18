package terminal

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"testing"
)

func readLine(input string) (string, string, error) {
	return readLineWithHistory(input)
}

func readLineWithHistory(input string, history ...string) (string, string, error) {
	in := bufio.NewReader(strings.NewReader(input))
	var out strings.Builder
	state := &ShellState{CommandHistory: history}
	line, err := ReadLine(in, &out, state)
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

func TestReadLineAppendsUnhandledControlChar(t *testing.T) {
	// A bare tab (0x09) is not handled specially, so the default case echoes
	// it through into the line verbatim.
	line, _, err := readLine("a\tb\r")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line != "a\tb" {
		t.Fatalf("expected %q, got %q", "a\tb", line)
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

func TestReadLineUpArrowRecallsHistory(t *testing.T) {
	// ESC [ A is the up arrow; it should recall the most recent command.
	line, _, err := readLineWithHistory("\x1b[A\r", "ls", "pwd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line != "pwd" {
		t.Fatalf("expected %q, got %q", "pwd", line)
	}
}

func TestReadLineDownArrowMovesForward(t *testing.T) {
	// Up twice clamps at the newest entry, then Down steps back to the oldest.
	line, _, err := readLineWithHistory("\x1b[A\x1b[A\x1b[B\r", "ls", "pwd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line != "ls" {
		t.Fatalf("expected %q, got %q", "ls", line)
	}
}

func TestReadLineSS3ArrowForm(t *testing.T) {
	// ESC O A is the application-cursor-keys form of the up arrow.
	line, _, err := readLineWithHistory("\x1bOA\r", "ls", "pwd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line != "pwd" {
		t.Fatalf("expected %q, got %q", "pwd", line)
	}
}

func TestReadLineArrowWithEmptyHistory(t *testing.T) {
	// With no history, the arrow is a no-op and the line stays empty.
	line, out, err := readLineWithHistory("\x1b[A\r")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line != "" {
		t.Fatalf("expected empty line, got %q", line)
	}
	if strings.Contains(out, "pwd") {
		t.Fatalf("did not expect any recalled command, got %q", out)
	}
}

func TestReadLineUpArrowReplacesTypedInput(t *testing.T) {
	// Typing "abc" then pressing Up erases the typed text and shows history.
	line, out, err := readLineWithHistory("abc\x1b[A\r", "ls", "pwd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line != "pwd" {
		t.Fatalf("expected %q, got %q", "pwd", line)
	}
	// The three typed chars must be erased with "\b \b" before redrawing.
	if !strings.Contains(out, "\b \b\b \b\b \b") {
		t.Fatalf("expected typed input to be erased, got %q", out)
	}
}

func TestReadLineIgnoresNonArrowEscapeSequence(t *testing.T) {
	// ESC [ C is the right arrow, which is not handled and should be dropped
	// without disturbing surrounding input.
	line, _, err := readLineWithHistory("ab\x1b[Cx\r", "ls", "pwd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line != "abx" {
		t.Fatalf("expected %q, got %q", "abx", line)
	}
}

func TestReadLineEscapeAtEOF(t *testing.T) {
	// A lone ESC with no following bytes surfaces the reader's io.EOF.
	line, _, err := readLine("\x1b")
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF, got %v", err)
	}
	if line != "" {
		t.Fatalf("expected empty line, got %q", line)
	}
}
