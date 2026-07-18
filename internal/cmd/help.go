package cmd

import (
	"fmt"
	"sort"
	"strings"

	"goscouter/internal/style"
)

type HelpCommand struct {
	Manager *Manager
}

func (cmd *HelpCommand) Name() string {
	return "help"
}

func (cmd *HelpCommand) Description() string {
	return "Returns list of avaiable built-in commands."
}

func (cmd *HelpCommand) Exec(args []string) error {
	var b strings.Builder

	b.WriteString(style.Bold("Available commands") + "\r\n")
	if cmd.Manager != nil {
		names := make([]string, 0, len(cmd.Manager.Commands))
		width := 0
		for name, command := range cmd.Manager.Commands {
			if _, ok := command.(*ExternalCommand); ok {
				continue
			}
			names = append(names, name)
			if len(name) > width {
				width = len(name)
			}
		}
		sort.Strings(names)

		for _, name := range names {
			command := cmd.Manager.Commands[name]
			pad := strings.Repeat(" ", width-len(name))
			b.WriteString(fmt.Sprintf("  %s%s   %s\r\n",
				style.Cyan(name), pad, style.Dim(command.Description())))
		}
	}

	b.WriteString("\r\n" + style.Bold("Flags") + "\r\n")
	b.WriteString(fmt.Sprintf("  %s   %s\r\n",
		style.Cyan("--target"),
		style.Dim("Determines the site that goscouter will target (requires http/https prefix)."),
	))

	b.WriteString(fmt.Sprintf("  %s  %s\r\n",
		style.Cyan("--version"),
		style.Dim("Returns goscouter cli version"),
	))

    fmt.Printf("%s\r\n", b.String())
	return nil
}
