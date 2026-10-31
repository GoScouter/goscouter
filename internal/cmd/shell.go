package cmd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"
)

const ShellPrefix = "!"

func RunShell(ctx context.Context, line string) error {
	if line == "" {
		return errors.New("no shell command given")
	}

	var shell *exec.Cmd
	if runtime.GOOS == "windows" {
		shell = exec.CommandContext(ctx, "cmd", "/C", line)
	} else {
		shell = exec.CommandContext(ctx, "sh", "-c", line)
	}

	shell.Stdout = os.Stdout
	shell.Stderr = os.Stderr
	killTree(shell)

	err := shell.Run()
	if ctx.Err() != nil {
		return errors.New("interrupted")
	}
	return err
}
