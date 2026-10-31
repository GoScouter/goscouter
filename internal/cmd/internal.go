package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

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

func (cmd *InternalCommand) Flags() []Flag {
	if f, ok := cmd.Module.(module.FlagModule); ok {
		return f.Flags()
	}
	return nil
}

func (cmd *InternalCommand) Usage() string {
	usage := cmd.Name()
	for _, f := range cmd.Flags() {
		usage += " [" + f.Name + "]"
	}
	return usage
}

func (cmd *InternalCommand) Exec(ctx context.Context, args []string) error {
	data, views, err := module.RunInOrder(ctx, cmd.Module.Info(), cmd.CmdManager.Target, args, cmd.ModuleManager, cmd.Reporter)
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
