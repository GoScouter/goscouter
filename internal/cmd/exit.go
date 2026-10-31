package cmd

import (
	"context"
	"errors"
)

type ExitCommand struct{}

func (cmd *ExitCommand) Name() string {
	return "exit"
}

func (cmd *ExitCommand) Description() string {
	return "Exit's gs shell."
}

var ErrExit = errors.New("exit shell")

func (cmd *ExitCommand) Exec(ctx context.Context, args []string) error {
	return ErrExit
}
