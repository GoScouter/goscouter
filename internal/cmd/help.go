package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/GoScouter/sdk/style"
)

type HelpCommand struct {
	Manager *CommandManager
}

func (cmd *HelpCommand) Name() string {
	return "help"
}

func (cmd *HelpCommand) Description() string {
	return "Returns list of avaiable built-in commands."
}

func (cmd *HelpCommand) Usage() string {
	return "help [<command>]"
}

func (cmd *HelpCommand) CompleteArgs(args []string, word string) []string {
	if len(args) > 0 || cmd.Manager == nil {
		return nil
	}
	return cmd.Manager.completeCommand(word)
}

func (cmd *HelpCommand) Exec(ctx context.Context, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("usage: %s", cmd.Usage())
	}
	if len(args) == 1 {
		return cmd.describe(args[0])
	}

	var b strings.Builder

	b.WriteString(style.Bold("Available commands") + "\r\n")
	if cmd.Manager != nil {
		names := make([]string, 0, len(cmd.Manager.Commands))
		width := 0
		for name := range cmd.Manager.Commands {
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

	b.WriteString(fmt.Sprintf("  %s  %s\r\n",
		style.Cyan("--install"),
		style.Dim("Installs an external module"),
	))

	b.WriteString(fmt.Sprintf("  %s  %s\r\n",
		style.Cyan("--uninstall"),
		style.Dim("Uninstalls an external module"),
	))

	b.WriteString("\r\n" + style.Bold("Keys") + "\r\n")
	for _, k := range keys {
		b.WriteString(fmt.Sprintf("  %s%s   %s\r\n",
			style.Cyan(k[0]), strings.Repeat(" ", keysWidth-len([]rune(k[0]))), style.Dim(k[1])))
	}

	b.WriteString("\r\n" + style.Dim("Run ") + style.Cyan("help <command>") + style.Dim(" for a command's usage and flags.") + "\r\n")

	fmt.Printf("%s\r\n", b.String())
	return nil
}

const keysWidth = 14

var keys = [][2]string{
	{"Tab / →", "Complete the command or argument"},
	{"↑ / ↓", "Browse history"},
	{"Ctrl-R", "Search history"},
	{"Ctrl-F", "Search the output"},
	{"Ctrl-C", "Cancel the running command"},
	{"!<command>", "Run a system command, e.g. !ls -la"},
	{"Mouse wheel", "Scroll"},
	{"Drag", "Select and copy"},
	{"Click prompt", "Fold or unfold that command's output"},
	{"Ctrl+click", "Open a link"},
}

func (cmd *HelpCommand) describe(name string) error {
	if cmd.Manager == nil {
		return fmt.Errorf("no commands to describe")
	}
	target, err := cmd.Manager.Get(name)
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString(style.Bold(style.Cyan(target.Name())) + "  " + style.Dim(target.Description()) + "\r\n\r\n")
	b.WriteString(style.Bold("Usage") + "\r\n  " + usage(target) + "\r\n")

	if short := shortName(target.Name()); short != target.Name() {
		b.WriteString(style.Dim("  (or just "+short+")") + "\r\n")
	}

	if fs := flags(target); len(fs) > 0 {
		width := 0
		for _, f := range fs {
			width = max(width, len(f.Name))
		}
		b.WriteString("\r\n" + style.Bold("Flags") + "\r\n")
		for _, f := range fs {
			b.WriteString(fmt.Sprintf("  %s%s   %s\r\n",
				style.Cyan(f.Name), strings.Repeat(" ", width-len(f.Name)), style.Dim(f.Description)))
		}
	}

	fmt.Printf("%s\r\n", b.String())
	return nil
}
