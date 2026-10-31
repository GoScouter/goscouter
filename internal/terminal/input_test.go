package terminal

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"testing"
)

var commands = []string{"clear", "exit", "help", "info", "internal:dns", "internal:subdomains", "target"}

func complete(line string) []string {
	if cmd, word, ok := strings.Cut(line, " "); ok {
		if cmd != "target" || strings.Contains(word, " ") {
			return nil
		}
		var hosts []string
		for _, h := range []string{"example.com", "websec.co.il"} {
			if strings.HasPrefix(h, word) {
				hosts = append(hosts, h)
			}
		}
		return hosts
	}

	word := line
	var full, short []string
	for _, name := range commands {
		if strings.HasPrefix(name, word) {
			full = append(full, name)
		} else if _, s, ok := strings.Cut(name, ":"); ok && strings.HasPrefix(s, word) {
			short = append(short, s)
		}
	}
	return append(full, short...)
}

func readLine(input string) (string, string, error) {
	return readLineWithHistory(input)
}

func readLineWithHistory(input string, history ...string) (string, string, error) {
	in := bufio.NewReader(strings.NewReader(input))
	var out strings.Builder
	state := &ShellState{CommandHistory: history}
	line, err := ReadLine(in, &out, state, LineOptions{Prompt: "> ", Complete: complete})
	return line, out.String(), err
}

func expectLine(t *testing.T, input, want string, history ...string) {
	t.Helper()
	line, _, err := readLineWithHistory(input, history...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line != want {
		t.Fatalf("input %q: expected %q, got %q", input, want, line)
	}
}

func TestReadLineCarriageReturn(t *testing.T) {
	expectLine(t, "hello\r", "hello")
}

func TestReadLineNewline(t *testing.T) {
	expectLine(t, "world\n", "world")
}

func TestReadLinePrintsPrompt(t *testing.T) {
	_, out, _ := readLine("\r")
	if !strings.Contains(out, "> ") {
		t.Fatalf("expected the prompt in the output, got %q", out)
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

func TestReadLineCtrlDDeletesForward(t *testing.T) {
	// Ctrl-A to the start, then Ctrl-D deletes the character under the cursor.
	expectLine(t, "hi\x01\x04\r", "i")
}

func TestReadLineBackspace(t *testing.T) {
	expectLine(t, "ab\x7fc\r", "ac")
}

func TestReadLineBackspaceOnEmptyLine(t *testing.T) {
	expectLine(t, "\x7f\r", "")
}

func TestReadLineIgnoresControlChars(t *testing.T) {
	expectLine(t, "a\x07b\r", "ab")
}

func TestReadLineEchoesInput(t *testing.T) {
	_, out, _ := readLine("xy\r")
	if !strings.Contains(out, "xy") {
		t.Fatalf("expected echoed input %q, got %q", "xy", out)
	}
}

func TestReadLineErrorPropagates(t *testing.T) {
	line, _, err := readLine("partial")
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF, got %v", err)
	}
	if line != "partial" {
		t.Fatalf("expected buffered %q returned with error, got %q", "partial", line)
	}
}

func TestReadLineEscapeAtEOF(t *testing.T) {
	line, _, err := readLine("\x1b")
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF, got %v", err)
	}
	if line != "" {
		t.Fatalf("expected empty line, got %q", line)
	}
}

func TestReadLineCursorMovement(t *testing.T) {
	cases := []struct {
		name, input, want string
	}{
		{"left then insert", "ac\x1b[Db\r", "abc"},
		{"right moves back", "abc\x1b[D\x1b[D\x1b[CX\r", "abXc"},
		{"home", "bc\x1b[Ha\r", "abc"},
		{"end", "ab\x1b[H\x1b[Fc\r", "abc"},
		{"home tilde form", "bc\x1b[1~a\r", "abc"},
		{"ctrl-a and ctrl-e", "b\x01a\x05c\r", "abc"},
		{"ctrl-b and ctrl-f", "ac\x02\x02\x06b\r", "abc"},
		{"delete key", "abc\x1b[H\x1b[3~\r", "bc"},
		{"ctrl-k", "abcd\x02\x02\x0b\r", "ab"},
		{"ctrl-u", "abcd\x02\x02\x15\r", "cd"},
		{"ctrl-w", "run some words\x17\r", "run some "},
		{"ctrl-w skips trailing spaces", "run some  \x17\r", "run "},
		{"alt-key ignored", "a\x1bxb\r", "ab"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			expectLine(t, c.input, c.want)
		})
	}
}

func TestReadLineHistory(t *testing.T) {
	history := []string{"one", "two", "three"}
	cases := []struct {
		name, input, want string
	}{
		{"up recalls newest", "\x1b[A\r", "three"},
		{"up twice", "\x1b[A\x1b[A\r", "two"},
		{"up stops at oldest", "\x1b[A\x1b[A\x1b[A\x1b[A\r", "one"},
		{"down moves forward", "\x1b[A\x1b[A\x1b[B\r", "three"},
		{"down restores typed text", "draft\x1b[A\x1b[B\r", "draft"},
		{"down without browsing", "x\x1b[B\r", "x"},
		{"ss3 form", "\x1bOA\r", "three"},
		{"ctrl-p and ctrl-n", "\x10\x10\x0e\r", "three"},
		{"recalled entry can be edited", "\x1b[A!\r", "three!"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			expectLine(t, c.input, c.want, history...)
		})
	}
}

func TestReadLineArrowWithEmptyHistory(t *testing.T) {
	expectLine(t, "\x1b[A\r", "")
}

func TestReadLineHistoryNotModifiedByEditing(t *testing.T) {
	history := []string{"scan"}
	expectLine(t, "\x1b[A\x7f\x7f\r", "sc", history...)
	if history[0] != "scan" {
		t.Fatalf("editing a recalled entry changed the history to %q", history[0])
	}
}

func TestReadLineTabCompletion(t *testing.T) {
	cases := []struct {
		name, input, want string
	}{
		{"unique full name", "he\t\r", "help "},
		{"unique short name", "subdo\t\r", "subdomains "},
		{"common prefix", "in\t\r", "in"},
		{"extends to common prefix", "inte\t\r", "internal:"},
		{"no match", "zzz\t\r", "zzz"},
		{"not after the command", "target sub\t\r", "target sub"},
		{"not for shell commands", "!he\t\r", "!he"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			expectLine(t, c.input, c.want)
		})
	}
}

func TestReadLineTabListsAmbiguousMatches(t *testing.T) {
	_, out, _ := readLine("in\t\r")
	for _, name := range []string{"info", "internal:dns", "internal:subdomains"} {
		if !strings.Contains(out, name) {
			t.Fatalf("expected %q in the listed matches, got %q", name, out)
		}
	}
}

func TestReadLineRightArrowAcceptsSuggestion(t *testing.T) {
	expectLine(t, "subdo\x1b[C\r", "subdomains")
}

func TestReadLineShowsSuggestion(t *testing.T) {
	_, out, _ := readLine("subdo\x03")
	if !strings.Contains(out, "mains") {
		t.Fatalf("expected the suggestion %q in the output, got %q", "mains", out)
	}
}

func TestReadLineSuggestionNotSubmitted(t *testing.T) {
	expectLine(t, "subdo\r", "subdo")
}

func TestAddHistorySkipsRepeats(t *testing.T) {
	state := &ShellState{}
	state.AddHistory("help")
	state.AddHistory("help")
	state.AddHistory("")
	state.AddHistory("scan")
	if got := strings.Join(state.CommandHistory, ","); got != "help,scan" {
		t.Fatalf("expected history %q, got %q", "help,scan", got)
	}
}
