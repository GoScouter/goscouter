package cmd

import (
	"fmt"

	"github.com/GoScouter/sdk"
)

type ExternalCommand struct {
	Manager    *Manager
	ModuleName string
	Module     string
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
	defer bin.Close()

	res, err := bin.Scout(cmd.Manager.Target, args)
	if err != nil {
		return err
	}

	fmt.Println(res.Render())
	return nil
}
