package cmd

import (
	"fmt"

	"github.com/GoScouter/sdk"
)

type ModuleCommand struct {
	Manager *Manager
	Module  sdk.Module
}

func (cmd *ModuleCommand) Name() string {
	return cmd.Module.Name()
}

func (cmd *ModuleCommand) Description() string {
	return cmd.Module.Description()
}

func (cmd *ModuleCommand) Exec(args []string) error {
	result, err := cmd.Module.Scout(cmd.Manager.Target, args)
	if err != nil {
		return err
	}

	fmt.Print(result.Render())
	return nil
}
