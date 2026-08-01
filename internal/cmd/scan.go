package cmd

import (
	"fmt"
	"goscouter/internal/module"
	"goscouter/internal/scanning"
)

type ScanCommand struct {
	CmdManager    *CommandManager
	ModuleManager *module.Manager
}

func (cmd *ScanCommand) Name() string {
	return "scan"
}

func (cmd *ScanCommand) Description() string {
	return "Makes a complete scan of the target"
}

func (cmd *ScanCommand) Exec(args []string) error {
	output, err := scanning.Scan(cmd.CmdManager.Target, cmd.ModuleManager)
	if err != nil {
		return err
	}

	fmt.Print("\r\n" + scanning.Render(output))
	return nil
}
