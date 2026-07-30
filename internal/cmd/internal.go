package cmd

import (
	"fmt"
	"goscouter/internal/module"

	"github.com/GoScouter/sdk"
)

type InternalCommand struct {
	Manager *CommandManager
	Module  sdk.Module
}

func (cmd *InternalCommand) Name() string {
	return module.Key(module.Namespace(cmd.Module.Info()))
}

func (cmd *InternalCommand) Description() string {
	return cmd.Module.Info().Description
}

func (cmd *InternalCommand) Exec(args []string) error {
	data, err := cmd.Module.Scout(cmd.Manager.Target, args)
	if err != nil {
		return err
	}

	fmt.Println(cmd.Module.Render(data))
	return nil
}
