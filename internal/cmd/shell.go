package cmd

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
)

// ShellPrefix marks a line as a system shell command, e.g. "!ls -la".
const ShellPrefix = "!"

// RunShell runs line with the system shell, wired to the current terminal.
func RunShell(line string) error {
	if line == "" {
		return errors.New("no shell command given")
	}

	var shell *exec.Cmd
	if runtime.GOOS == "windows" {
		shell = exec.Command("cmd", "/C", line)
	} else {
		shell = exec.Command("sh", "-c", line)
	}

	shell.Stdin = os.Stdin
	shell.Stdout = os.Stdout
	shell.Stderr = os.Stderr

	return shell.Run()
}
