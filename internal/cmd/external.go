package cmd

import (
    "fmt"

    "github.com/GoScouter/sdk"
)

type ExternalCommand struct {
    Target string
    ModuleName string
    Module string
}

func (cmd *ExternalCommand) Name() string {
    return cmd.ModuleName
}

func (cmd *ExternalCommand) Description() string {
    return "No need :O"
}

func (cmd *ExternalCommand) Exec(args []string) error {
    bin, err := sdk.Open(cmd.Module)
    if err != nil {
        return err
    }

    res, err := bin.Scout(cmd.Target)
    if err != nil {
        return err
    }

    fmt.Println(res.Render())
    return nil
}
