//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

// detach puts the child in a session of its own, so it survives its parent. A new session
// was measured not to cost a daemon its GUI session on macOS (B23), which is the reason the
// plugin's shell function takes either branch.
func detach(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }
