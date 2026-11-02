//go:build !windows

package adapters

import (
	"os/exec"
	"syscall"
)

// killProcessGroupOnCancel makes context cancellation kill the whole group:
// `uv run` spawns python as a child, and killing uv alone orphans it holding GPU memory.
func killProcessGroupOnCancel(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
}
