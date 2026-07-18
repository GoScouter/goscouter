package cmd

import (
	"fmt"

	"goscouter/internal/logger"
	"goscouter/internal/style"
)

type TargetCommand struct {
	Manager *Manager
}

func (cmd *TargetCommand) Name() string {
	return "target"
}

func (cmd *TargetCommand) Description() string {
	return "Shows or sets the site that goscouter targets"
}

func (cmd *TargetCommand) Exec(args []string) error {
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

	logger.Log.Info(fmt.Sprintf("Target set to %q", target))
	fmt.Printf("%s\r\n", style.Successf("Target set to %s", style.Bold(target)))
	return nil
}
