//go:build unix

package cmd

import (
	"os/exec"
	"syscall"
)

// killTree makes cancelling kill everything the shell started, e.g. both
// sides of "ping x | grep y", not just the shell itself.
func killTree(shell *exec.Cmd) {
	shell.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	shell.Cancel = func() error {
		return syscall.Kill(-shell.Process.Pid, syscall.SIGKILL)
	}
}
