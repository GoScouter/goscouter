package cmd

import (
	"fmt"
	"strings"

	"goscouter/internal/module"

	"github.com/GoScouter/sdk"
)

type InternalCommand struct {
	CmdManager    *CommandManager
	ModuleManager *module.Manager
	Module        sdk.Module

	Reporter *module.Reporter
}

func (cmd *InternalCommand) Name() string {
	return module.Key(module.Namespace(cmd.Module.Info()))
}

func (cmd *InternalCommand) Description() string {
	return cmd.Module.Info().Description
}

func (cmd *InternalCommand) Exec(args []string) error {
	_, views, err := module.RunInOrder(cmd.Module.Info(), cmd.CmdManager.Target, args, cmd.ModuleManager, cmd.Reporter)
	if err != nil {
		return err
	}

	fmt.Print(strings.Join(views, ""))
	return nil
}
