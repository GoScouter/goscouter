package style

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

const (
	reset = "\033[0m"

	codeBold = "\033[1m"
	codeDim  = "\033[2m"

	codeRed    = "\033[38;2;235;77;75m"
	codeGreen  = "\033[38;2;111;207;151m"
	codeYellow = "\033[38;2;249;202;54m"
	codeCyan   = "\033[38;2;56;193;208m"
	codeGray   = "\033[38;2;130;130;150m"
	codePurple = "\033[38;2;87;87;232m"
	codeWhite  = "\033[38;2;255;255;255m"
)

var enabled = detect()

func detect() bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func wrap(code, s string) string {
	if !enabled {
		return s
	}
	return code + s + reset
}

func Bold(s string) string   { return wrap(codeBold, s) }

// BoldAll makes an already-styled string bold across every color segment.
// Each color helper ends its span with a reset, which would also clear bold, so
// bold is re-asserted after each reset instead of just wrapping the whole line.
func BoldAll(s string) string {
	if !enabled {
		return s
	}
	return codeBold + strings.ReplaceAll(s, reset, reset+codeBold) + reset
}

func Dim(s string) string    { return wrap(codeDim, s) }
func Red(s string) string    { return wrap(codeRed, s) }
func Green(s string) string  { return wrap(codeGreen, s) }
func Yellow(s string) string { return wrap(codeYellow, s) }
func Cyan(s string) string   { return wrap(codeCyan, s) }
func Gray(s string) string   { return wrap(codeGray, s) }
func Purple(s string) string { return wrap(codePurple, s) }
func White(s string) string  { return wrap(codeWhite, s) }

func Prompt() string {
	return Dim("(") + Bold(Purple("gs")) + Dim(")") + " " + Cyan("❯") + " "
}

func Error(msg string) string {
	return Red("✗ ") + msg
}

func Errorf(format string, a ...any) string {
	return Error(fmt.Sprintf(format, a...))
}

func Success(msg string) string {
	return Green("✓ ") + msg
}

func Successf(format string, a ...any) string {
	return Success(fmt.Sprintf(format, a...))
}

func Info(msg string) string {
	return Cyan("» ") + msg
}

func Infof(format string, a ...any) string {
	return Info(fmt.Sprintf(format, a...))
}
