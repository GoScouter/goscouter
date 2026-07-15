package cmd

import (
	"fmt"

	"goscouter/internal/logger"
	"goscouter/internal/module"
	"goscouter/internal/style"
)

type InstallCommand struct {
	Manager *Manager
	Target  string
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

	link := args[0]
	if link == "" {
		return fmt.Errorf("usage: install <module-ref>")
	}

	logger.Log.Info(fmt.Sprintf("Installing module from %q\r\n", link))

	ref := module.ParseModule(link)
	if ref == nil {
		fmt.Printf("%s\r\n", style.Infof("Resolving manifest from %s", link))
		manifest, err := module.ResolveManifest(link)
		if err != nil {
			return err
		}

		binaryPath, err := module.Download(manifest)
		if err != nil {
			return err
		}

		return cmd.register(binaryPath)
	}

	fmt.Printf("%s\r\n", style.Infof("Resolving module %s", ref.ToString()))

	// Make sure to make the registry website to have an api endpoint that doing this.
	// Needs a domain for that because github pages cannot do this.
	const rawBase = "https://raw.githubusercontent.com/GoScouter/registry/main"
	url := fmt.Sprintf("%s/%s/%s/%s/manifest.json", rawBase, ref.Author, ref.Module, ref.Version)

	logger.Log.Info(fmt.Sprintf("Fetching manifest from %s", url))
	manifest, err := module.ResolveManifest(url)
	if err != nil {
		return err
	}

	if ref.Module != manifest.Name {
		return fmt.Errorf("module name mismatch (%s != %s)", ref.Module, manifest.Name)
	}

	if ref.Version != manifest.Version {
		return fmt.Errorf("module version mismatch (%s != %s)", ref.Version, manifest.Version)
	}

	binaryPath, err := module.Download(manifest)
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
		Target:     cmd.Target,
		ModuleName: name,
		Module:     binaryPath,
	})

	fmt.Printf("%s\r\n", style.Successf("Command %s is now available", style.Bold(name)))
	logger.Log.Info(fmt.Sprintf("Registered external command %q from %s", name, binaryPath))
	return nil
}
