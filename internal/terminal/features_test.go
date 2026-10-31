package terminal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReadLineHistorySearch(t *testing.T) {
	history := []string{"target example.com", "dns", "target websec.co.il", "scan"}
	cases := []struct {
		name, input, want string
	}{
		{"finds newest match", "\x12target\r", "target websec.co.il"},
		{"ctrl-r again finds older", "\x12target\x12\r", "target example.com"},
		{"ignores case", "\x12SCAN\r", "scan"},
		{"backspace widens the search", "\x12dnx\x7f\r", "dns"},
		{"esc restores the line", "typed\x12dns\x07\r", "typed"},
		{"right arrow edits the match", "\x12dns\x1b[C!\r", "dns!"},
		{"no match keeps the last match", "\x12dns\x12\r", "dns"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			expectLine(t, c.input, c.want, history...)
		})
	}
}

func TestReadLineHistorySearchShowsQuery(t *testing.T) {
	_, out, _ := readLineWithHistory("\x12zzz\x03", "dns")
	if !strings.Contains(out, "no match in history") {
		t.Fatalf("expected the failed search to show, got %q", out)
	}
}

func TestReadLineCompletesArguments(t *testing.T) {
	expectLine(t, "target web\t\r", "target websec.co.il ")
	expectLine(t, "target web\x1b[C\r", "target websec.co.il")
}

func TestReadLineArgumentGhost(t *testing.T) {
	_, out, _ := readLine("target exa\x03")
	if !strings.Contains(out, "mple.com") {
		t.Fatalf("expected the argument suggestion, got %q", out)
	}
}

func TestReadLineSendsHint(t *testing.T) {
	s, _ := testScreen(60, 3, "")
	state := &ShellState{Screen: s}

	var asked []string
	_, _ = ReadLine(bufioReader(strings.NewReader("dns")), &strings.Builder{}, state, LineOptions{
		Prompt: "> ",
		Hint: func(line string) string {
			asked = append(asked, line)
			return "hint for " + line
		},
	})

	if len(asked) == 0 || asked[len(asked)-1] != "dns" {
		t.Fatalf("expected the editor to ask for a hint for %q, asked for %q", "dns", asked)
	}
	if s.status.hint != "" {
		t.Fatalf("expected the hint cleared after reading, got %q", s.status.hint)
	}
}

func TestStatusBar(t *testing.T) {
	s, _ := testScreen(80, 3, "")
	s.SetTarget("websec.co.il")

	bar, _ := s.statusBar()
	if !strings.Contains(bar, "websec.co.il") || !strings.Contains(bar, "Ctrl-R history") {
		t.Fatalf("expected the target and tips when idle, got %q", bar)
	}

	s.Busy("internal:scan")
	bar, _ = s.statusBar()
	if !strings.Contains(bar, "internal:scan") || !strings.Contains(bar, "Ctrl-C cancel") {
		t.Fatalf("expected the running command, got %q", bar)
	}
	s.Idle()

	bar, _ = s.statusBar()
	if strings.Contains(bar, "internal:scan") {
		t.Fatalf("expected the command gone after Idle, got %q", bar)
	}
}

func TestStatusBarFitsWidth(t *testing.T) {
	s, _ := testScreen(20, 3, "")
	s.SetTarget("a-very-long-target-name.example.com")
	bar, _ := s.statusBar()
	if w := len([]rune(ansiRE.ReplaceAllString(bar, ""))); w != 20 {
		t.Fatalf("expected the bar to be exactly 20 columns, got %d: %q", w, bar)
	}
}

func TestFormatElapsed(t *testing.T) {
	cases := map[time.Duration]string{
		4 * time.Second:                "4s",
		65 * time.Second:               "1m05s",
		12*time.Minute + 3*time.Second: "12m03s",
	}
	for d, want := range cases {
		if got := formatElapsed(d); got != want {
			t.Errorf("formatElapsed(%v) = %q, want %q", d, got, want)
		}
	}
}

func TestOutputSearch(t *testing.T) {
	s, _ := testScreen(40, 3, numbered(30)+"\n")
	events(s, "\x06")
	if s.search == nil {
		t.Fatal("expected Ctrl-F to open the search")
	}

	events(s, "line 1")
	// "line 1", "line 10".."line 19": 11 matches, newest selected.
	if n := len(s.search.matches); n != 11 {
		t.Fatalf("expected 11 matches, got %d", n)
	}
	if s.search.current != 10 {
		t.Fatalf("expected the newest match selected, got %d", s.search.current)
	}

	events(s, "\r\r")
	if s.search.current != 8 {
		t.Fatalf("expected Enter to step to older matches, got %d", s.search.current)
	}
	if got := strings.Join(visible(s), "|"); !strings.Contains(got, "line 17") {
		t.Fatalf("expected the view on the match, got %q", got)
	}

	events(s, "\x1b[B")
	if s.search.current != 9 {
		t.Fatalf("expected ↓ to step to a newer match, got %d", s.search.current)
	}

	if got := keysRead(s); got != "" {
		t.Fatalf("search keys must not reach the editor, got %q", got)
	}
}

func TestOutputSearchCloses(t *testing.T) {
	s, _ := testScreen(40, 3, numbered(30)+"\n")
	r := bufioReader(strings.NewReader("\x06line 2\x1b"))
	for s.readEvent(r) == nil {
	}
	if s.search != nil {
		t.Fatal("expected Esc to close the search")
	}
	if s.offset != 0 {
		t.Fatalf("expected closing to return to the bottom, got offset %d", s.offset)
	}
}

func TestOutputSearchHighlights(t *testing.T) {
	s, tty := testScreen(40, 3, "find me here\n> ")
	events(s, "\x06me")
	s.render()
	if !strings.Contains(tty.String(), "\x1b[30;43mme") {
		t.Fatalf("expected the match highlighted, got %q", tty.String())
	}
}

func commandOutput() string {
	return "> first\r\n" + BlockMarker + "one\na\nb\n" +
		"> second\r\n" + BlockMarker + "two\n" +
		"> "
}

func TestBufferRecordsBlocks(t *testing.T) {
	b := bufferOf(commandOutput())
	if len(b.blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(b.blocks))
	}
	if got := lineText(b, b.blocks[0].header); got != "> first" {
		t.Fatalf("expected the first block to start at its prompt, got %q", got)
	}
	if got := lineText(b, b.blocks[1].header); got != "> second" {
		t.Fatalf("expected the second block to start at its prompt, got %q", got)
	}
	if strings.Contains(lineText(b, 1), "goscouter") {
		t.Fatal("the marker must not show up as text")
	}
}

func TestFoldBlock(t *testing.T) {
	s, _ := testScreen(60, 10, commandOutput())

	// Click the first prompt line.
	events(s, "\x1b[<0;3;1M\x1b[<0;3;1m")
	got := strings.Join(visible(s), "|")
	if got != "> first|> second|two|> " {
		t.Fatalf("expected the first block folded, got %q", got)
	}

	s.render()
	if frame := strings.Join(s.lastFrame, ""); !strings.Contains(frame, "3 lines folded") {
		t.Fatalf("expected a folded note, got %q", frame)
	}

	// Click it again to unfold.
	events(s, "\x1b[<0;3;1M\x1b[<0;3;1m")
	if got := strings.Join(visible(s), "|"); got != "> first|one|a|b|> second|two|> " {
		t.Fatalf("expected the block unfolded, got %q", got)
	}
}

func TestFoldLastBlockOnlyWhenIdle(t *testing.T) {
	s, _ := testScreen(60, 10, commandOutput())

	s.Busy("second")
	s.mu.Lock()
	folded := s.toggleFold(s.buf.blocks[1].header)
	s.mu.Unlock()
	if folded {
		t.Fatal("a running command's output must not fold")
	}
	s.Idle()

	s.mu.Lock()
	folded = s.toggleFold(s.buf.blocks[1].header)
	s.mu.Unlock()
	if !folded {
		t.Fatal("expected the last command to fold once it finished")
	}
	if got := strings.Join(visible(s), "|"); !strings.HasSuffix(got, "> second|> ") {
		t.Fatalf("expected the prompt to stay visible, got %q", got)
	}
}

func TestSearchUnfolds(t *testing.T) {
	s, _ := testScreen(60, 10, commandOutput())
	events(s, "\x1b[<0;3;1M\x1b[<0;3;1m") // fold "first"
	events(s, "\x06one")
	if s.buf.blocks[0].folded {
		t.Fatal("expected a search match to unfold its block")
	}
}

func TestFindLinks(t *testing.T) {
	b := bufferOf("see https://example.com/a?b=1, and http://x.io.")
	links := findLinks(b.lines[0])
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}
	if links[0].url != "https://example.com/a?b=1" || links[1].url != "http://x.io" {
		t.Fatalf("got %+v", links)
	}
	if links[0].start != 4 {
		t.Fatalf("expected the first link at column 4, got %d", links[0].start)
	}
}

func TestRenderLinks(t *testing.T) {
	s, tty := testScreen(60, 3, "go to https://example.com now\n> ")
	s.render()
	if !strings.Contains(tty.String(), linkStart("https://example.com")) {
		t.Fatalf("expected an OSC 8 link, got %q", tty.String())
	}
}

func TestURLAt(t *testing.T) {
	b := bufferOf("go to https://example.com now")
	if got := urlAt(b, pos{line: 0, col: 10}); got != "https://example.com" {
		t.Fatalf("got %q", got)
	}
	if got := urlAt(b, pos{line: 0, col: 1}); got != "" {
		t.Fatalf("expected no link outside the URL, got %q", got)
	}
}

func TestLoneEscapeReachesEditorAsCancel(t *testing.T) {
	s, _ := testScreen(20, 3, "")
	r := bufioReader(strings.NewReader("\x1b"))
	_ = s.readEvent(r)
	if got := keysRead(s); got != "\x07" {
		t.Fatalf("expected Esc to arrive as Ctrl-G, got %q", got)
	}
}

func TestHistoryFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gs", "history")

	if h, err := loadHistory(path); err != nil || h != nil {
		t.Fatalf("expected an empty history for a missing file, got %v, %v", h, err)
	}

	state := &ShellState{historyPath: path}
	state.AddHistory("dns")
	state.AddHistory("dns")
	state.AddHistory("scan")

	got, err := loadHistory(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "dns,scan" {
		t.Fatalf("expected the saved history %q, got %q", "dns,scan", got)
	}
}

func TestHistoryFileTrimmed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history")
	var lines []string
	for i := range maxHistory + 50 {
		lines = append(lines, "cmd"+string(rune('a'+i%26)))
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	history, _ := loadHistory(path)
	if len(history) > maxHistory {
		history = history[len(history)-maxHistory:]
		_ = saveHistory(path, history)
	}

	saved, _ := loadHistory(path)
	if len(saved) != maxHistory {
		t.Fatalf("expected the file cut to %d entries, got %d", maxHistory, len(saved))
	}
}

func TestStatusBarPinned(t *testing.T) {
	s, tty := testScreen(40, 5, "> ")

	if !strings.Contains(tty.String(), "\x1b[1;5r") {
		t.Fatalf("expected a scroll region above the status bar, got %q", tty.String())
	}

	tty.mu.Lock()
	tty.b.Reset()
	tty.mu.Unlock()
	s.write([]byte("x"))
	s.render()
	if !strings.Contains(tty.String(), "\x1b[6;1H") {
		t.Fatalf("expected the status bar redrawn, got %q", tty.String())
	}
}

func TestRowsNeverWiderThanScreen(t *testing.T) {
	s, _ := testScreen(10, 3, "0123456789abcdefghij\n> ")
	s.render()
	for y, line := range s.lastFrame {
		plain := ansiRE.ReplaceAllString(line, "")
		if w := len([]rune(plain)); w > 10 {
			t.Fatalf("row %d is %d columns wide: %q", y, w, plain)
		}
	}
}

func TestShorten(t *testing.T) {
	cases := []struct {
		in   string
		n    int
		want string
	}{
		{"example.com", 32, "example.com"},
		{"abcdefghij", 5, "ab…ij"},
		{"abcdefghij", 6, "ab…hij"},
		{"abcdefghij", 1, "a"},
	}
	for _, c := range cases {
		if got := shorten(c.in, c.n); got != c.want {
			t.Errorf("shorten(%q, %d) = %q, want %q", c.in, c.n, got, c.want)
		}
	}
}

func TestStatusBarLongTarget(t *testing.T) {
	s, _ := testScreen(120, 3, "")
	s.SetTarget(strings.Repeat("a", 200) + ".example.com")

	bar, _ := s.statusBar()
	plain := ansiRE.ReplaceAllString(bar, "")
	if !strings.Contains(plain, "…") || !strings.HasSuffix(strings.Fields(plain)[1], ".example.com") {
		t.Fatalf("expected the target shortened in the middle, got %q", plain)
	}
	if !strings.Contains(plain, "Ctrl-R history") {
		t.Fatalf("expected the tips to stay visible, got %q", plain)
	}
}

func TestDomainRoot(t *testing.T) {
	cases := map[string]string{
		"example.com":            ".com",
		"https://websec.co.il":   ".co.il",
		"shop.example.com.au":    ".com.au",
		"www.abc.com":            ".com",
		"api.example.io":         ".io",
		"localhost":              "",
		"https://sub.example.co": ".co",
	}
	for host, want := range cases {
		if got := domainRoot(host); got != want {
			t.Errorf("domainRoot(%q) = %q, want %q", host, got, want)
		}
	}
}

func TestShortenTargetKeepsRoot(t *testing.T) {
	long := strings.Repeat("a", 50)
	cases := []struct {
		in, root string
		n        int
	}{
		{"https://" + long + ".co.il", ".co.il", 20},
		{long + ".com", ".com", 12},
		{"http://" + long + ".example.com.au/", ".com.au", 16},
	}
	for _, c := range cases {
		got := shortenTarget(c.in, c.n)
		if !strings.HasSuffix(got, c.root) {
			t.Errorf("shortenTarget(%q, %d) = %q, want it to end in %q", c.in, c.n, got, c.root)
		}
		if n := len([]rune(got)); n > c.n {
			t.Errorf("shortenTarget(%q, %d) = %q, %d characters", c.in, c.n, got, n)
		}
		if !strings.Contains(got, "…") || strings.Contains(got, "://") {
			t.Errorf("shortenTarget(%q, %d) = %q, want it shortened without the scheme", c.in, c.n, got)
		}
	}
}

func TestShortenTargetFits(t *testing.T) {
	if got := shortenTarget("https://websec.co.il", 32); got != "https://websec.co.il" {
		t.Fatalf("got %q", got)
	}
	if got := shortenTarget("https://websec.co.il", 12); got != "websec.co.il" {
		t.Fatalf("got %q", got)
	}
	if got := shortenTarget(strings.Repeat("c", 50)+".co.il", 7); got != "….co.il" {
		t.Fatalf("got %q", got)
	}
}

func TestOldSelectionNotCopiedAgain(t *testing.T) {
	s, tty := testScreen(40, 3, "see https://example.com here\nother\n> ")
	events(s, "\x1b[<0;1;1M\x1b[<32;3;1M\x1b[<0;3;1m")

	// A Ctrl+click and a click on the status bar both end with a release.
	events(s, "\x1b[<16;10;1M\x1b[<16;10;1m")
	events(s, "\x1b[<0;5;4M\x1b[<0;5;4m")

	if n := strings.Count(tty.String(), "\x1b]52;"); n != 1 {
		t.Fatalf("expected the selection copied once, got %d copies", n)
	}
}

func TestCloseInputEndsReadLine(t *testing.T) {
	s, _ := testScreen(20, 3, "")
	s.CloseInput()
	_, err := ReadLine(bufioReader(s.Input()), &strings.Builder{}, &ShellState{Screen: s}, LineOptions{Prompt: "> "})
	if err == nil {
		t.Fatal("expected ReadLine to stop once the input is closed")
	}
}
