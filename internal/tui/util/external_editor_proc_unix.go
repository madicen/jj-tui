//go:build !windows

package util

import (
	"os/exec"
	"syscall"
)

func startDetached(cmd *exec.Cmd) error {
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd.Start()
}
