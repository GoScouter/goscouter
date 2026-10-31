//go:build !unix && !windows

package cmd

import "os/exec"

func killTree(*exec.Cmd) {}
