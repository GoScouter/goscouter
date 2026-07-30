package cmd

import (
	"context"
	"fmt"
	"goscouter/internal/module"
)

type ExternalCommand struct {
	Manager  *CommandManager
	External *module.External
}

func (cmd *ExternalCommand) Name() string {
	return module.Key(module.Namespace(cmd.External.Info))
}

func (cmd *ExternalCommand) Description() string {
	return cmd.External.Info.Description
}

func (cmd *ExternalCommand) Exec(args []string) error {
	_, view, err := cmd.External.Scout(context.Background(), cmd.Manager.Target, args)
	if err != nil {
		return err
	}

	fmt.Println(view)
	return nil
}
