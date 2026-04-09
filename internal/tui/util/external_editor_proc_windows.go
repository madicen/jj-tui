//go:build windows

package util

import "os/exec"

func startDetached(cmd *exec.Cmd) error {
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}
