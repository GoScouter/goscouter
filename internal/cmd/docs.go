package cmd

import (
	"slices"
	"strings"

	"goscouter/internal/module"
)

type Flag = module.Flag

type Usager interface {
	Usage() string
}

type Flagger interface {
	Flags() []Flag
}

type ArgCompleter interface {
	CompleteArgs(args []string, word string) []string
}

func usage(cmd Command) string {
	if u, ok := cmd.(Usager); ok {
		return u.Usage()
	}
	return cmd.Name()
}

func flags(cmd Command) []Flag {
	if f, ok := cmd.(Flagger); ok {
		return f.Flags()
	}
	return nil
}

func (cm *CommandManager) Complete(line string) []string {
	i := strings.LastIndexByte(line, ' ')
	if i < 0 {
		return cm.completeCommand(line)
	}

	fields := strings.Fields(line[:i])
	if len(fields) == 0 {
		return nil
	}
	cmd, err := cm.Get(fields[0])
	if err != nil {
		return nil
	}
	args, word := fields[1:], line[i+1:]

	var cands []string
	if ac, ok := cmd.(ArgCompleter); ok {
		cands = ac.CompleteArgs(args, word)
	}
	if word == "" || strings.HasPrefix(word, "-") {
		for _, f := range flags(cmd) {
			if strings.HasPrefix(f.Name, word) && !slices.Contains(args, f.Name) {
				cands = append(cands, f.Name)
			}
		}
	}
	return cands
}

// completeCommand returns the commands that word could be the start of,
// matching either the full name or the part after the namespace. A short-name
// match completes to the short name ("subdo" -> "subdomains"), which Get
// resolves, unless another command shares that short name; then it completes
// to the full name so it stays unambiguous. Full-name matches come first.
func (cm *CommandManager) completeCommand(word string) []string {
	shortNames := make(map[string]int)
	for name := range cm.Commands {
		shortNames[shortName(name)]++
	}

	var full, short []string
	for name := range cm.Commands {
		if strings.HasPrefix(name, word) {
			full = append(full, name)
		} else if s := shortName(name); s != name && strings.HasPrefix(s, word) {
			if shortNames[s] == 1 {
				short = append(short, s)
			} else {
				short = append(short, name)
			}
		}
	}
	slices.Sort(full)
	slices.Sort(short)
	return append(full, short...)
}

func (cm *CommandManager) Hint(line string) string {
	fields := strings.Fields(line)
	if len(fields) == 0 || strings.HasPrefix(line, ShellPrefix) {
		return ""
	}
	cmd, err := cm.Get(fields[0])
	if err != nil {
		return ""
	}
	return usage(cmd) + "  ·  " + cmd.Description()
}

func shortName(name string) string {
	if _, s, ok := strings.Cut(name, ":"); ok {
		return s
	}
	return name
}
