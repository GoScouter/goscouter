package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

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

func (cmd *ExternalCommand) Exec(ctx context.Context, args []string) error {
	if cmd.Reporter == nil {
		return fmt.Errorf("cannot use %s command, socket conn was not found", cmd.Name())
	}

	data, views, err := module.RunInOrder(ctx, cmd.External.Info, cmd.CmdManager.Target, args, cmd.ModuleManager, cmd.Reporter)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	cmd.CmdManager.Last = &Result{
		Command: cmd.Name(),
		Target:  cmd.CmdManager.Target,
		Time:    time.Now(),
		Data:    data,
		Views:   views,
	}

	fmt.Print(strings.Join(views, ""))
	return nil
}
