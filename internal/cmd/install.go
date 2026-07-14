package cmd

import (
	"fmt"
	"goscouter/internal/module"
)

type InstallCommand struct {}

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
        return fmt.Errorf("Usage: install <moudle-ref>")
    }

    ref := module.ParseModule(link)
    if ref == nil {
        _, err := module.ResolveManifest(link)
        if err != nil {
            return err
        }

        fmt.Printf("TODO: Download binary (non-official)")
        return nil
    }

    // Make sure to make the registry website to have an api endpoint that doing this.
    // Needs a domain for that because github pages cannot do this.
    const rawBase = "https://raw.githubusercontent.com/GoScouter/registry/main"
    url := fmt.Sprintf("%s/%s/%s/%s/manifest.json", rawBase, ref.Author, ref.Module, ref.Version)

    manifest, err := module.ResolveManifest(url)
    if err != nil {
        return err
    }

    fmt.Printf("%+v\r\n", manifest)
    fmt.Printf("TODO: Download binary (official)\r\n")

    return nil
}
