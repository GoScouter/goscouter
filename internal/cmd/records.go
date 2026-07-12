package cmd

import (
    "fmt"

	"goscouter/internal/module"
)

type RecordsCommand struct {
	Target string
	Module module.Module
}

func (cmd *RecordsCommand) Name() string {
	return "records"
}

func (cmd *RecordsCommand) Description() string {
	return "Show the DNS and HTTP records of the target website."
}

func (cmd *RecordsCommand) Exec(args []string) error {
	result, err := cmd.Module.Scout(cmd.Target)
	if err != nil {
		return err
	}

	fmt.Print(result.Render())
	return nil
}
