//go:build windows

package cmd

import (
	"os/exec"
	"strconv"
)

// killTree makes cancelling kill everything cmd started, not just cmd itself.
func killTree(shell *exec.Cmd) {
	shell.Cancel = func() error {
		return exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(shell.Process.Pid)).Run()
	}
}
