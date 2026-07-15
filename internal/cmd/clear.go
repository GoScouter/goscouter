package cmd

import (
	"fmt"
)

type ClearCommand struct{}

func (cmd *ClearCommand) Name() string {
	return "clear"
}

func (cmd *ClearCommand) Description() string {
	return "Clear's current buffer"
}

const clear = "\033[2J\033[H"

func (cmd *ClearCommand) Exec(args []string) error {
	fmt.Print(clear)
	return nil
}
