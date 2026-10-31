package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoScouter/sdk/style"
)

type TargetCommand struct {
	Manager *CommandManager
}

func (cmd *TargetCommand) Name() string {
	return "target"
}

func (cmd *TargetCommand) Description() string {
	return "Shows or sets the site that goscouter targets"
}

func (cmd *TargetCommand) Usage() string {
	return "target [<example.com>]"
}

func (cmd *TargetCommand) CompleteArgs(args []string, word string) []string {
	if len(args) > 0 {
		return nil
	}

	var history []string
	if cmd.Manager.History != nil {
		history = cmd.Manager.History()
	}

	seen := make(map[string]bool)
	var cands []string
	add := func(t string) {
		if t != "" && !seen[t] && strings.HasPrefix(t, word) {
			seen[t] = true
			cands = append(cands, t)
		}
	}
	for i := len(history) - 1; i >= 0; i-- {
		if fields := strings.Fields(history[i]); len(fields) == 2 && fields[0] == "target" {
			add(fields[1])
		}
	}
	add(cmd.Manager.Target)
	return cands
}

func (cmd *TargetCommand) Exec(ctx context.Context, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("usage: target [<example.com>]")
	}

	if len(args) == 0 {
		fmt.Printf("%s %s\r\n", style.Gray("Target:"), style.Bold(cmd.Manager.Target))
		return nil
	}

	target := args[0]
	if target == "" {
		return fmt.Errorf("usage: target [<example.com>]")
	}

	cmd.Manager.SetTarget(target)

	fmt.Printf("%s\r\n", style.Successf("Target set to %s", style.Bold(target)))
	return nil
}
