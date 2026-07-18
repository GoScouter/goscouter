package utils

import (
	"fmt"
	"strings"

	"goscouter/internal/style"
)

const BANNER = `
 ██████╗  ██████╗ ███████╗ ██████╗ ██████╗ ██╗   ██╗████████╗███████╗██████╗
██╔════╝ ██╔═══██╗██╔════╝██╔════╝██╔═══██╗██║   ██║╚══██╔══╝██╔════╝██╔══██╗
██║  ███╗██║   ██║███████╗██║     ██║   ██║██║   ██║   ██║   █████╗  ██████╔╝
██║   ██║██║   ██║╚════██║██║     ██║   ██║██║   ██║   ██║   ██╔══╝  ██╔══██╗
╚██████╔╝╚██████╔╝███████║╚██████╗╚██████╔╝╚██████╔╝   ██║   ███████╗██║  ██║
 ╚═════╝  ╚═════╝ ╚══════╝ ╚═════╝ ╚═════╝  ╚═════╝    ╚═╝   ╚══════╝╚═╝  ╚═╝
`

const NAME = "GS"

func Banner(version, buildTime string) string {
	var b strings.Builder

	lines := strings.Split(BANNER, "\n")
	for _, line := range lines {
		fmt.Fprintf(&b, "%s\r\n", style.Purple(line))
	}
	fmt.Fprintln(&b)

	fmt.Fprintf(&b, "\t\t%s %s %s %s\r\n\r\n",
		style.Bold(NAME),
		style.Cyan(version),
		style.Dim("•"),
		style.Dim(buildTime),
	)

	return b.String()
}

func PrintBanner(version, buildTime string) {
	fmt.Print(Banner(version, buildTime))
}
