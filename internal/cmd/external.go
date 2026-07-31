package cmd

import (
	"fmt"
	"goscouter/internal/module"
)

type ExternalCommand struct {
	CmdManager    *CommandManager
	ModuleManager *module.Manager
	External      *module.External

	Reporter *module.Reporter
}

func (cmd *ExternalCommand) Name() string {
	return module.Key(module.Namespace(cmd.External.Info))
}

func (cmd *ExternalCommand) Description() string {
	return cmd.External.Info.Description
}

func (cmd *ExternalCommand) Exec(args []string) error {
	if cmd.Reporter == nil {
		return fmt.Errorf("cannot use %s command, socket conn was not found", cmd.Name())
	}

	_, view, err := module.RunInOrder(cmd.External.Info, cmd.CmdManager.Target, args, cmd.ModuleManager, cmd.Reporter)
	if err != nil {
		return err
	}

	fmt.Print(view)
	return nil
}
