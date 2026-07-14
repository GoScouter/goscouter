package cmd

import (
	"fmt"

	"goscouter/internal/logger"
	"goscouter/internal/module"
)

type UninstallCommand struct {
	Manager *Manager
}

func (cmd *UninstallCommand) Name() string {
	return "uninstall"
}

func (cmd *UninstallCommand) Description() string {
	return "Uninstalls a module"
}

func (cmd *UninstallCommand) Exec(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: uninstall <module-name>")
	}

	name := args[0]
	if name == "" {
		return fmt.Errorf("usage: uninstall <module-name>")
	}

	logger.Log.Info(fmt.Sprintf("Uninstalling module %q", name))

	if _, err := module.Remove(name + execSuffix()); err != nil {
		return err
	}

	return cmd.unregister(name)
}

func (cmd *UninstallCommand) unregister(name string) error {
	if cmd.Manager == nil {
		return nil
	}

	cmd.Manager.Remove(name)

	fmt.Printf("Command %q is no longer available\r\n", name)
	logger.Log.Info(fmt.Sprintf("Unregistered external command %q", name))
	return nil
}
