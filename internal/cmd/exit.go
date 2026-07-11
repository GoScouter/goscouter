package cmd

import (
	"errors"
)

type ExitCommand struct {}

func (cmd *ExitCommand) Name() string {
    return "exit"
}

var ErrExit = errors.New("exit shell")

func (cmd *ExitCommand) Exec(args []string) error {
    return ErrExit
}

