package cmd

import (
	"fmt"

	"goscouter/internal/logger"
	"goscouter/internal/module"
	"goscouter/internal/style"
)

type InstallCommand struct {
	Manager *Manager
}

func (cmd *InstallCommand) Name() string {
	return "install"
}

func (cmd *InstallCommand) Description() string {
	return "Installs a module"
}

func (cmd *InstallCommand) Exec(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: install <module-ref>")
	}

	url := args[0]
	if url == "" {
		return fmt.Errorf("usage: install <module-ref>")
	}

	logger.Log.Info(fmt.Sprintf("Installing module from %q\r\n", url))
	ref := module.ParseModule(url)
    manifest, err := module.ResolveManifest(ref)
	if err != nil {
		return err
	}

	binaryPath, err := module.Download(manifest, ref.Version)
	if err != nil {
		return err
	}

	return cmd.register(binaryPath)
}

func (cmd *InstallCommand) register(binaryPath string) error {
	if cmd.Manager == nil {
		return nil
	}

	name := commandName(binaryPath)
	cmd.Manager.Add(&ExternalCommand{
		Manager:    cmd.Manager,
		ModuleName: name,
		Module:     binaryPath,
	})

	fmt.Printf("%s\r\n", style.Successf("Command %s is now available", style.Bold(name)))
	logger.Log.Info(fmt.Sprintf("Registered external command %q from %s", name, binaryPath))
	return nil
}
