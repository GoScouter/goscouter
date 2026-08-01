package style

import (
	"fmt"
	"os"
	"regexp"
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

func Bold(s string) string { return wrap(codeBold, s) }

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

func rawLines(msg string) string {
	msg = strings.ReplaceAll(msg, "\r\n", "\n")
	return strings.ReplaceAll(msg, "\n", "\r\n")
}

func Error(msg string) string {
	return Red("✗ ") + rawLines(msg)
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

var ansiRE = regexp.MustCompile("\x1b\\[[0-9;]*m")

func Width(s string) int {
	return len([]rune(ansiRE.ReplaceAllString(s, "")))
}

func Found(msg string) string {
	return Green("[+] ") + msg
}

func Foundf(format string, a ...any) string {
	return Found(fmt.Sprintf(format, a...))
}

func Failure(msg string) string {
	return Red("[-] ") + msg
}

func Failuref(format string, a ...any) string {
	return Failure(fmt.Sprintf(format, a...))
}

func Alert(msg string) string {
	return Yellow("[!] ") + rawLines(msg)
}

func Alertf(format string, a ...any) string {
	return Alert(fmt.Sprintf(format, a...))
}

func Section(title string, body ...string) string {
	var b strings.Builder

	b.WriteString("\r\n")
	b.WriteString(Bold(Cyan("["+title+"]")) + "\r\n")
	for _, line := range body {
		b.WriteString(line + "\r\n")
	}
	b.WriteString("\r\n")

	return b.String()
}

func Field(label string, width int, value string) string {
	return "  " + Gray(fmt.Sprintf("%-*s", width, label)) + " " + value
}
