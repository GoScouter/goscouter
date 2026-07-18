package cmd

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"

	"goscouter/internal/style"
	"goscouter/internal"
)

type InfoCommand struct{}

func (cmd *InfoCommand) Name() string {
	return "info"
}

func (cmd *InfoCommand) Description() string {
	return "Shows general information about the tool"
}

var logo = []string{
	style.Cyan("           --==============--"),
	style.Cyan("  .-==-.===oooo=oooooo=ooooo===--===-"),
	style.Cyan(" .==  =o=") + style.White("oGGGGGG") + style.Cyan("o=oo=o") + style.Green("GGGGGGG") + style.Cyan("G=o=") + style.Green(" ╱") + style.Red("◉"),
	style.Cyan(" -o= oo=") + style.White("G .=GGGGG") + style.Cyan("o=o=") + style.Green("= .=GGGGG") + style.Cyan("=ooo") + style.Green("══▊"),
	style.Cyan("  .-=oo=") + style.White("o==oGGGGG") + style.Cyan("=oo=") + style.Green("oooGGGGGo") + style.Cyan("=oooo."),
	style.Cyan("   -ooooo") + style.White("=oooooo") + style.Cyan("=") + style.Yellow(".   .") + style.Cyan("=") + style.White("=ooo==") + style.Cyan("oooooo-"),
	style.Cyan("   -ooooooooooo") + style.Yellow("====_====") + style.Cyan("ooooooooooo="),
	style.Cyan("   -oooooooooooo") + style.Yellow("==") + style.White("#") + style.Cyan(".") + style.White("#") + style.Yellow("==") + style.Cyan("ooooooooooooo"),
	style.Cyan("   -ooooooooooooo=") + style.White("#") + style.Cyan(".") + style.White("#") + style.Cyan("=oooooooooooooo"),
	style.Cyan("   .oooooooooooooooooooooooooooooooo."),
	style.Cyan("    oooooooooooooooooooooooooooooooo."),
	style.Yellow("  ..") + style.Cyan("oooooooooooooooooooooooooooooooo") + style.Yellow(".."),
	style.Yellow("-=o-") + style.Cyan("=ooooooooooooooooooooooooooooooo") + style.Yellow("-oo."),
	style.Yellow(".=- ") + style.Cyan("oooooooooooooooooooooooooooooooo") + style.Yellow("-.-"),
	style.Cyan("   .oooooooooooooooooooooooooooooooo-"),
	style.Cyan("   -oooooooooooooooooooooooooooooooo-"),
	style.Cyan("   -oooooooooooooooooooooooooooooooo-"),
	style.Cyan("   -oooooooooooooooooooooooooooooooo-"),
	style.Cyan("   .oooooooooooooooooooooooooooooooo"),
	style.Cyan("    =oooooooooooooooooooooooooooooo-"),
	style.Cyan("    .=oooooooooooooooooooooooooooo-"),
	style.Cyan("      -=oooooooooooooooooooooooo=."),
	style.Yellow("     =oo") + style.Cyan("====oooooooooooooooo==-") + style.Yellow("oo=-"),
	style.Yellow("    .-==-    ") + style.Cyan(".--=======---     ") + style.Yellow(".==-"),
}

var ansiRE = regexp.MustCompile("\x1b\\[[0-9;]*m")

func visibleWidth(s string) int {
	return len([]rune(ansiRE.ReplaceAllString(s, "")))
}

func field(key, value string) string {
	return style.Bold(style.White(key)) + style.Gray(" : ") + value
}

func colorSwatch() string {
	const block = "███"
	return style.Red(block) + style.Yellow(block) + style.Green(block) +
		style.Cyan(block) + style.Purple(block) + style.White(block)
}

func infoLines() []string {
	title := style.Bold(style.Cyan("GoScouter"))
	rule := style.Gray(strings.Repeat("─", visibleWidth(title)+9))

	return []string{
		title,
		rule,
		"",
		field("GitHub  ", style.Green("github.com/GoScouter/goscouter")),
		field("Website ", style.Cyan("https://goscouter.github.io/")),
		"",
		field("Version ", style.Green(internal.Version)),
		field("Go      ", style.Yellow(strings.TrimPrefix(runtime.Version(), "go"))),
		field("System  ", style.Yellow(runtime.GOOS+" / "+runtime.GOARCH)),
		field("License ", style.Yellow("GPL-3.0")),
		"",
		style.Bold(style.White("Purpose")),
		style.Gray("  • Scouting"),
		style.Gray("  • Network probing"),
		style.Gray("  • Enumeration"),
		style.Gray("  • Analysis"),
		"",
		colorSwatch(),
	}
}

func (cmd *InfoCommand) Exec(args []string) error {
	info := infoLines()

	logoWidth := 0
	for _, line := range logo {
		if w := visibleWidth(line); w > logoWidth {
			logoWidth = w
		}
	}

	const gap = 3
	rows := len(logo)
	if len(info) > rows {
		rows = len(info)
	}

	var b strings.Builder
	b.WriteString("\r\n")
	for i := 0; i < rows; i++ {
		logoLine := ""
		if i < len(logo) {
			logoLine = style.BoldAll(logo[i])
		}

        padding := logoWidth + gap - visibleWidth(logoLine)
		if padding < 0 {
			padding = 0
		}

        b.WriteString(logoLine)
		b.WriteString(strings.Repeat(" ", padding))

		if i < len(info) {
			b.WriteString(info[i])
		}

        b.WriteString("\r\n")
	}
	b.WriteString("\r\n")

	fmt.Print(b.String())
	return nil
}
